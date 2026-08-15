APP_NAME    := MoleBar
BIN_NAME    := molebar
BUILD_DIR   := build
DIST_DIR    := dist
APP_BUNDLE  := $(BUILD_DIR)/$(APP_NAME).app
CONTENTS    := $(APP_BUNDLE)/Contents

# Authoritative version is the git tag (v0.1.2 → 0.1.2). Override with
# VERSION=... for a release job that already knows the tag.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
ifeq ($(strip $(VERSION)),)
VERSION := dev
endif

MACOSX_DEPLOYMENT_TARGET ?= 11.0
SDKROOT ?= $(shell xcrun --sdk macosx --show-sdk-path 2>/dev/null)
GOFLAGS := -buildvcs=false -trimpath -ldflags="-s -w"

# Optional Developer ID identity. Local builds do not require this.
CODESIGN_IDENTITY ?=

.PHONY: all build build-native build-universal app test vet check fmt clean run dist sign

all: app

## build: compile the raw binary (native architecture).
build: build-native

build-native:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		go build $(GOFLAGS) -o $(BUILD_DIR)/$(BIN_NAME) ./cmd/molebar

## build-universal: arm64 + amd64 via lipo. Requires a macOS SDK that can
## target both architectures. Do not claim a universal binary unless lipo succeeds.
build-universal:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		go build $(GOFLAGS) -o $(BUILD_DIR)/$(BIN_NAME)-arm64 ./cmd/molebar
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		CC="clang -arch x86_64$(if $(SDKROOT), -isysroot $(SDKROOT))" \
		CGO_CFLAGS="$(if $(SDKROOT),-isysroot $(SDKROOT))" \
		CGO_LDFLAGS="$(if $(SDKROOT),-isysroot $(SDKROOT))" \
		go build $(GOFLAGS) -o $(BUILD_DIR)/$(BIN_NAME)-amd64 ./cmd/molebar
	lipo -create -output $(BUILD_DIR)/$(BIN_NAME) \
		$(BUILD_DIR)/$(BIN_NAME)-arm64 $(BUILD_DIR)/$(BIN_NAME)-amd64
	lipo -info $(BUILD_DIR)/$(BIN_NAME)
	file $(BUILD_DIR)/$(BIN_NAME)

## app: package the binary into a .app bundle. VERSION is stamped into Info.plist.
## Set UNIVERSAL=1 to require a universal binary (no native fallback).
app:
	@if [ "$(UNIVERSAL)" = "1" ]; then \
		$(MAKE) build-universal; \
	else \
		$(MAKE) build-native; \
	fi
	mkdir -p $(CONTENTS)/MacOS $(CONTENTS)/Resources
	cp $(BUILD_DIR)/$(BIN_NAME) $(CONTENTS)/MacOS/$(BIN_NAME)
	sed "s|__VERSION__|$(VERSION)|g" packaging/Info.plist > $(CONTENTS)/Info.plist
	cp packaging/MoleBar.icns $(CONTENTS)/Resources/MoleBar.icns
	@$(MAKE) sign
	@echo "Built $(APP_BUNDLE) (version $(VERSION)) — drag it into /Applications, then launch it."

sign:
ifeq ($(CODESIGN_IDENTITY),)
	@echo "Skipping codesign (CODESIGN_IDENTITY unset). See README for Developer ID / notarization setup."
else
	codesign --force --options runtime --sign "$(CODESIGN_IDENTITY)" --timestamp "$(APP_BUNDLE)"
	codesign --verify --verbose "$(APP_BUNDLE)"
endif

## run: build and run directly in the terminal.
run: build
	./$(BUILD_DIR)/$(BIN_NAME)

test:
	go test ./...

vet:
	go vet ./...

## check: verify formatting and run tests without modifying files.
check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt'd:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	go mod verify
	go test ./...
	go vet ./...

fmt:
	gofmt -l -w .

## dist: release artifacts only (no .git, no working-tree junk).
## Uses a universal binary when the SDK can target both architectures.
dist:
	$(MAKE) app UNIVERSAL=1
	mkdir -p $(DIST_DIR)
	ditto -c -k --keepParent --norsrc --noextattr --noqtn "$(APP_BUNDLE)" "$(DIST_DIR)/$(APP_NAME)-$(VERSION).app.zip"
	git archive --format=tar.gz --prefix=$(BIN_NAME)-$(VERSION)/ -o "$(DIST_DIR)/$(BIN_NAME)-$(VERSION).tar.gz" HEAD
	@echo "Wrote $(DIST_DIR)/$(APP_NAME)-$(VERSION).app.zip and $(DIST_DIR)/$(BIN_NAME)-$(VERSION).tar.gz"

clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
