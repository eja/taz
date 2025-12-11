.PHONY: clean test lint taz android-libs

PACKAGE_NAME := github.com/eja/taz
GOLANG_CROSS_VERSION := v1.22.2
GOPATH ?= '$(HOME)/go'

all: lint taz

clean:
	@rm -rf build

lint:
	@gofmt -w ./app

taz:
	@mkdir -p build
	@CGO_ENABLED=0 go build -ldflags "-s -w" -o build/taz ./app

android-libs:
	@GOOS=android GOARCH=arm64 go build -ldflags "-s -w" -o android/app/src/main/jniLibs/arm64-v8a/libtaz.so ./app
