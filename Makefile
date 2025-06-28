.PHONY: clean test lint taz

PACKAGE_NAME := github.com/eja/taz
GOLANG_CROSS_VERSION := v1.22.2
GOPATH ?= '$(HOME)/go'

all: taz

clean:
	@rm -f taz taz.exe

lint:
	@gofmt -w .

taz:
	@go build -ldflags "-s -w" -o taz .
	@strip taz
