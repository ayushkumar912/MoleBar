package alerts

import "time"

const defaultCooldown = 5 * time.Minute

// Alert is a rule in a reportable state.
type Alert struct {
	Rule  Rule
	State State
	Value float64
	Since time.Time
}

// AlertEvent is a state-transition notification. The engine never
// delivers macOS notifications itself.
type AlertEvent struct {
	Rule  Rule
	State State
	Value float64
	At    time.Time
}

type ruleState struct {
	state      State
	since      time.Time
	hits       int
	lastNotify time.Time
	lastValue  float64
}

// Engine evaluates rules against explicit samples. It is not safe for
// concurrent use; the application event loop owns it.
type Engine struct {
	rules    []Rule
	states   map[string]*ruleState
	cooldown time.Duration
}

// NewEngine constructs an engine. Invalid rules are ignored at
// evaluation time rather than rejected here so tests can assert that.
func NewEngine(rules []Rule, cooldown time.Duration) *Engine {
	if cooldown < 0 {
		cooldown = defaultCooldown
	}
	copied := make([]Rule, len(rules))
	copy(copied, rules)
	return &Engine{
		rules:    copied,
		states:   make(map[string]*ruleState, len(rules)),
		cooldown: cooldown,
	}
}

// Evaluate advances every rule using values sampled at at.
// Missing metrics cannot fire; a previously firing rule recovers.
// Only transitions produce events.
func (e *Engine) Evaluate(at time.Time, values map[Metric]float64) []AlertEvent {
	if e == nil {
		return nil
	}
	var events []AlertEvent
	for _, rule := range e.rules {
		if err := rule.Validate(); err != nil {
			continue
		}
		st := e.state(rule)
		val, ok := values[rule.Metric]
		if !ok {
			events = append(events, e.missing(at, rule, st)...)
			continue
		}
		events = append(events, e.step(at, rule, st, val)...)
	}
	return events
}

func (e *Engine) state(rule Rule) *ruleState {
	id := rule.id()
	st, ok := e.states[id]
	if !ok {
		st = &ruleState{state: StateInactive}
		e.states[id] = st
	}
	return st
}

func (e *Engine) missing(at time.Time, rule Rule, st *ruleState) []AlertEvent {
	if st.state == StateFiring || st.state == StatePending {
		prev := st.state
		st.state = StateRecovered
		st.hits = 0
		st.since = at
		if prev == StateFiring {
			return []AlertEvent{{Rule: rule, State: StateRecovered, Value: st.lastValue, At: at}}
		}
	}
	if st.state != StateRecovered {
		st.state = StateInactive
	}
	return nil
}

func (e *Engine) step(at time.Time, rule Rule, st *ruleState, val float64) []AlertEvent {
	st.lastValue = val
	crossed := rule.holds(val)
	switch st.state {
	case StateInactive, StateRecovered:
		if !crossed {
			st.state = StateInactive
			st.hits = 0
			return nil
		}
		st.state = StatePending
		st.since = at
		st.hits = 1
		if e.readyToFire(at, rule, st) {
			return e.fire(at, rule, st, val)
		}
		return nil
	case StatePending:
		if !crossed {
			st.state = StateInactive
			st.hits = 0
			return nil
		}
		st.hits++
		if e.readyToFire(at, rule, st) {
			return e.fire(at, rule, st, val)
		}
		return nil
	case StateFiring:
		if crossed {
			return nil
		}
		st.state = StateRecovered
		st.hits = 0
		st.since = at
		return []AlertEvent{{Rule: rule, State: StateRecovered, Value: val, At: at}}
	default:
		st.state = StateInactive
		return nil
	}
}

func (e *Engine) readyToFire(at time.Time, rule Rule, st *ruleState) bool {
	if st.hits < 2 {
		return false
	}
	if at.Sub(st.since) < rule.Duration {
		return false
	}
	if e.cooldown > 0 && !st.lastNotify.IsZero() && at.Sub(st.lastNotify) < e.cooldown {
		return false
	}
	return true
}

func (e *Engine) fire(at time.Time, rule Rule, st *ruleState, val float64) []AlertEvent {
	st.state = StateFiring
	st.lastNotify = at
	return []AlertEvent{{Rule: rule, State: StateFiring, Value: val, At: at}}
}

// Firing returns rules currently in the firing state.
func (e *Engine) Firing() []Alert {
	if e == nil {
		return nil
	}
	var out []Alert
	for _, rule := range e.rules {
		st, ok := e.states[rule.id()]
		if !ok || st.state != StateFiring {
			continue
		}
		out = append(out, Alert{Rule: rule, State: st.state, Value: st.lastValue, Since: st.since})
	}
	return out
}

// Reset clears evaluation state without changing the rule list.
func (e *Engine) Reset() {
	if e == nil {
		return
	}
	e.states = make(map[string]*ruleState, len(e.rules))
}
