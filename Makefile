NDK_ROOT := /Users/sean/Library/Android/sdk/ndk/26.1.10909125/
APP_ROOT := /Users/sean/AndroidStudioProjects/DroidCamera/

build-binary:
	GOOS=android GOARCH=arm64 CGO_ENABLED=1 \
		CGO_CFLAGS="-I$(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/sysroot/usr/include -I$(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/sysroot/usr/include/aarch64-linux-android" \
    	CGO_LDFLAGS="-L$(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/sysroot/usr/lib" \
        CC=$(shell realpath $(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android30-clang) \
        go build -v -tags localcgo,no_cgo \
        -o bin/droidcamera-android-aarch64 \
        ./cmd

build-mobile:
	gomobile bind -v -target android -androidapi 29 -o bin/droidcam.aar ./camera/

push-binary:
	adb push bin/droidcamera-android-aarch64 /data/local/tmp/

push-asset:
	cp bin/droidcamera-android-aarch64 $(APP_ROOT)/app/src/main/assets/droidcamera-android-aarch64

logs:
	adb logcat -s camera

