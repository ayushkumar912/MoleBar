package alerts

import "fmt"

// Notification is a platform-agnostic alert payload.
type Notification struct {
	Title string
	Body  string
}

// NotificationFromEvent formats a user-visible notification.
// It does not send anything.
func NotificationFromEvent(ev AlertEvent) Notification {
	name := string(ev.Rule.Metric)
	switch ev.State {
	case StateFiring:
		return Notification{
			Title: "MoleBar alert",
			Body:  fmt.Sprintf("%s %s %.0f (now %.1f)", name, ev.Rule.Operator, ev.Rule.Value, ev.Value),
		}
	case StateRecovered:
		return Notification{
			Title: "MoleBar recovered",
			Body:  fmt.Sprintf("%s is back within threshold", name),
		}
	default:
		return Notification{
			Title: "MoleBar",
			Body:  fmt.Sprintf("%s is %s", name, ev.State),
		}
	}
}
