.PHONY: all build run clean install deps

APP_NAME = MicMaxer2
APP_BUNDLE = $(APP_NAME).app

all: build

deps:
	go mod download
	go mod tidy

build: deps
	./build.sh

run: build
	open $(APP_BUNDLE)

clean:
	rm -rf $(APP_BUNDLE)
	go clean

install: build
	cp -r $(APP_BUNDLE) /Applications/
	@echo "$(APP_NAME) installed to /Applications/"

dev:
	go run .