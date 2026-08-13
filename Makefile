APP_NAME    := MoleBar
BIN_NAME    := molebar
BUILD_DIR   := build
APP_BUNDLE  := $(BUILD_DIR)/$(APP_NAME).app
CONTENTS    := $(APP_BUNDLE)/Contents

.PHONY: all build app clean run fmt vet

all: app

## build: compile the raw binary (no .app bundle, shows in Dock as a
## background process when run directly — mainly useful for `make run`).
build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(BIN_NAME) ./cmd/molebar

## app: package the binary into a proper .app bundle with LSUIElement set,
## so it launches from Spotlight/Applications and only appears in the menu
## bar (no Dock icon, no Cmd+Tab entry).
app: build
	mkdir -p $(CONTENTS)/MacOS
	cp $(BUILD_DIR)/$(BIN_NAME) $(CONTENTS)/MacOS/$(BIN_NAME)
	cp packaging/Info.plist $(CONTENTS)/Info.plist
	@echo "Built $(APP_BUNDLE) — drag it into /Applications, then launch it."

## run: build and run directly in the terminal (useful while developing;
## Ctrl+C to quit, or use the Quit item in the menu bar dropdown).
run: build
	./$(BUILD_DIR)/$(BIN_NAME)

fmt:
	gofmt -l -w .

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
