#ifndef CAMERA_NDK_H
#define CAMERA_NDK_H

#include <media/NdkImageReader.h>
#include <camera/NdkCameraDevice.h>
#include <camera/NdkCameraManager.h>
#include <pthread.h>
#include <android/log.h>

#ifdef __cplusplus
extern "C" {
#endif

#define TAG "camera"
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, TAG, __VA_ARGS__)
#define LOGW(...) __android_log_print(ANDROID_LOG_WARN, TAG, __VA_ARGS__)
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, TAG, __VA_ARGS__)

typedef struct {
    AImage* image;
    pthread_mutex_t mutex;
} GlobalImage;

typedef struct {
    bool cameraActive;
    bool cameraReady;
    bool cameraClosed;
    pthread_mutex_t mutex;
} CameraState;

extern GlobalImage globalImage;
extern CameraState cameraState;

extern AImage *image;
extern AImageReader *imageReader;

extern ANativeWindow *nativeWindow;

extern ACameraDevice *cameraDevice;
extern ACameraManager *cameraManager;
extern ACameraOutputTarget *cameraOutputTarget;
extern ACameraCaptureSession *cameraCaptureSession;

extern ACaptureRequest *captureRequest;
extern ACaptureSessionOutput *captureSessionOutput;
extern ACaptureSessionOutputContainer *captureSessionOutputContainer;

// Function declarations
void device_on_disconnected(void *context, ACameraDevice *device);
void device_on_error(void *context, ACameraDevice *device, int error);
void session_on_ready(void *context, ACameraCaptureSession *session);
void session_on_active(void *context, ACameraCaptureSession *session);
void session_on_closed(void *context, ACameraCaptureSession *session);
void image_callback(void *context, AImageReader *reader);
void freeImageBuff(void);
int openCamera(int index, int width, int height);
int captureCamera(void);
int closeCamera(void);

#ifdef __cplusplus
}
#endif

#endif // CAMERA_NDK_H