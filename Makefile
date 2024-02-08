NDK_ROOT := $(HOME)/Library/Android/sdk/ndk/26.1.10909125
APP_ROOT := $(HOME)/AndroidStudioProjects/DroidCamera

GOOS := android
GOARCH := arm64
CGO_ENABLED := 1
CC := $(shell realpath $(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android30-clang)
CGO_CFLAGS := -I$(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/sysroot/usr/include \
              -I$(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/sysroot/usr/include/aarch64-linux-android
CGO_LDFLAGS := -L$(NDK_ROOT)/toolchains/llvm/prebuilt/darwin-x86_64/sysroot/usr/lib
OUTPUT_DIR := bin
OUTPUT_NAME := droidcamera-android-aarch64
OUTPUT := $(OUTPUT_DIR)/$(OUTPUT_NAME)

ASSET_PATH := $(APP_ROOT)/app/src/main/assets/$(OUTPUT_NAME)
BINARY_PATH := /data/local/tmp/$(OUTPUT_NAME)

TARGET := android
ANDROID_API := 29
MOBILE_OUTPUT_NAME := droidcam.aar
MOBILE_OUTPUT := $(OUTPUT_DIR)/$(MOBILE_OUTPUT_NAME)

# Build the arm64 module binary
build-binary:
	@echo "Building binary for Android..."
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		CGO_CFLAGS="$(CGO_CFLAGS)" \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CC=$(CC) \
		go build -v -tags localcgo,no_cgo \
		-o $(OUTPUT) ./cmd
	@echo "Build complete: $(OUTPUT)"

# Build the mobile library
build-mobile:
	@echo "Building mobile library..."
	@gomobile bind -v -target $(TARGET) -androidapi $(ANDROID_API) -o $(MOBILE_OUTPUT) ./camera/
	@echo "Mobile library built: $(MOBILE_OUTPUT)"

# Push the binary to device
push-binary:
	@echo "Pushing binary to device..."
	@adb push $(OUTPUT) $(BINARY_PATH)
	@echo "Binary pushed: $(BINARY_PATH)"

# Copy the binary to project assets
push-asset:
	@echo "Copying binary to project assets..."
	@cp $(OUTPUT) $(ASSET_PATH)
	@echo "Binary copied to assets: $(ASSET_PATH)"

# Enable root access and set SELinux to permissive
root:
	@echo "Enabling root access and setting SELinux to permissive..."
	@adb root && adb shell "setenforce 0"
	@echo "Root access enabled and SELinux set to permissive."

# Filter logcat for camera logs
logs:
	@echo "Filtering logcat for camera logs..."
	@adb logcat -s camera
