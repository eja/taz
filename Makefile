.PHONY: clean test lint taz

PACKAGE_NAME := github.com/eja/taz
GOLANG_CROSS_VERSION := v1.22.2
GOPATH ?= '$(HOME)/go'

all: lint taz

clean:
	@rm -f taz taz.exe

lint:
	@gofmt -w ./app

taz:
	@CGO_ENABLED=0 go build -ldflags "-s -w" -o taz ./app
