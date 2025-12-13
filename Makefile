.PHONY: clean test lint taz android-libs

all: lint taz

clean:
	@rm -rf build

lint:
	@gofmt -w ./app

taz:
	@mkdir -p build
	@go build -ldflags "-s -w" -o build/taz ./app
	@GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o android/app/src/main/jniLibs/arm64-v8a/libtaz.so ./app
	@GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w" -o android/app/src/main/jniLibs/armeabi-v7a/libtaz.so ./app

