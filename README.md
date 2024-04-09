# Android Camera Modular Component

!!! warning
    This is a work in progress. The content is incomplete and subject to change.


## Build and Run

### Build and push the android-camera module

- Install the Android SDK and NDK.
- Follow instructions from rdk-apk to build rdk-server app and deploy to emulator/device.
    - Setup camera in emulator settings.
    - Enable camera permissions in the rdk-server app manually.
    - Run `make root` to get root access to the emulator and remove selinux restrictions.
- Run the following commands to build and push the android-camera module. (Make sure to replace hardcoded paths with your own paths.)

```bash
make build-binary
make push-binary
```

### Configure your camera

```json
    {
      "attributes": {
        "rotate": 90,
        "width": 640,
        "height": 480,
        "index": 0
      },
      "depends_on": [],
      "name": "droid-cam",
      "model": "viam:android:droid-cam",
      "type": "camera",
      "namespace": "rdk"
    }
```

```json
    {
      "executable_path": "/data/local/tmp/droidcamera-android-aarch64",
      "type": "local",
      "name": "droid-cam"
    }
```

### 32-bit arm builds

For testing only, do not use in production:

```sh
GOARCH=arm CC_ARCH=armv7a DROID_TARGET=androideabi30 make build-binary
```
