//go:build localcgo
// +build localcgo

// Package androidcamera.
package androidcamera

/*
#include <pthread.h>
#include <android/log.h>

#include <media/NdkImageReader.h>

#include <camera/NdkCameraDevice.h>
#include <camera/NdkCameraManager.h>

#define TAG "camera"
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, TAG, __VA_ARGS__)
#define LOGW(...) __android_log_print(ANDROID_LOG_WARN, TAG, __VA_ARGS__)
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, TAG, __VA_ARGS__)


// allow thread safetty for get_image callback
// and reads from golang runtime
typedef struct {
	AImage* image;
	pthread_mutex_t mutex;
} GlobalImage;

GlobalImage globalImage = {NULL, PTHREAD_MUTEX_INITIALIZER};

AImage *image;
AImageReader *imageReader;

ANativeWindow *nativeWindow;

ACameraDevice *cameraDevice;
ACameraManager *cameraManager;
ACameraOutputTarget *cameraOutputTarget;
ACameraCaptureSession *cameraCaptureSession;

ACaptureRequest *captureRequest;
ACaptureSessionOutput *captureSessionOutput;
ACaptureSessionOutputContainer *captureSessionOutputContainer;

void device_on_disconnected(void *context, ACameraDevice *device) {
    LOGI("camera %s is diconnected.\n", ACameraDevice_getId(device));
}

void device_on_error(void *context, ACameraDevice *device, int error) {
    LOGE("error %d on camera %s.\n", error, ACameraDevice_getId(device));
}

ACameraDevice_stateCallbacks deviceStateCallbacks = {
	.context = NULL,
	.onDisconnected = device_on_disconnected,
	.onError = device_on_error,
};

void session_on_ready(void *context, ACameraCaptureSession *session) {
    LOGI("session is ready. %p\n", session);
}

void session_on_active(void *context, ACameraCaptureSession *session) {
    LOGI("session is activated. %p\n", session);
}

void session_on_closed(void *context, ACameraCaptureSession *session) {
    LOGI("session is closed. %p\n", session);
}

ACameraCaptureSession_stateCallbacks captureSessionStateCallbacks = {
        .context = NULL,
        .onActive = session_on_active,
        .onReady = session_on_ready,
        .onClosed = session_on_closed,
};

void image_callback(void *context, AImageReader *reader) {
    LOGE("image_callback");

	AImage* img = NULL;
    media_status_t status = AImageReader_acquireLatestImage(reader, &img);
    if(status != AMEDIA_OK) {
		LOGE("failed to acquire next image (reason: %d).\n", status);
		return;
    }

	pthread_mutex_lock(&globalImage.mutex);
	if (globalImage.image != NULL) {
        AImage_delete(globalImage.image); // Free the old image
    }
	globalImage.image = img; // Update to the new image
    pthread_mutex_unlock(&globalImage.mutex);
}

// uses AImage_delete to free image
void freeImageBuff() {
    LOGI("free image buffer.\n");
    if(image != NULL) {
        AImage_delete(image);
        image = NULL;
    }
}


AImageReader_ImageListener imageListener = {
	.context = NULL,
	.onImageAvailable = image_callback,
};

int openCamera(int index, int width, int height) {
    LOGI("open camera (index: %d, width: %d, height: %d).\n", index, width, height);
    ACameraIdList *cameraIdList;
    const char *selectedCameraId;

    camera_status_t status = ACAMERA_OK;

    cameraManager = ACameraManager_create();

    status = ACameraManager_getCameraIdList(cameraManager, &cameraIdList);
    if(status != ACAMERA_OK) {
		LOGE("failed to get camera id list (reason: %d).\n", status);
		return status;
    }

    if(cameraIdList->numCameras < 1) {
		LOGE("no camera device detected.\n");
    }

    if(cameraIdList->numCameras < index+1) {
		LOGE("no camera at index %d.\n", index);
    }

    selectedCameraId = cameraIdList->cameraIds[index];
    LOGI("open camera (id: %s, num of cameras: %d).\n", selectedCameraId, cameraIdList->numCameras);

    status = ACameraManager_openCamera(cameraManager, selectedCameraId, &deviceStateCallbacks, &cameraDevice);
    if(status != ACAMERA_OK) {
		LOGE("failed to open camera device (id: %s)\n", selectedCameraId);
		return status;
    }
    LOGI("camera device opened.\n");

    status = ACameraDevice_createCaptureRequest(cameraDevice, TEMPLATE_STILL_CAPTURE, &captureRequest);
    if(status != ACAMERA_OK) {
		LOGE("failed to create snapshot capture request (id: %s)\n", selectedCameraId);
		return status;
    }
    LOGI("capture request created.\n");

    status = ACaptureSessionOutputContainer_create(&captureSessionOutputContainer);
    if(status != ACAMERA_OK) {
		LOGE("failed to create session output container (id: %s)\n", selectedCameraId);
		return status;
    }
    LOGI("session output container created.\n");

    media_status_t mstatus = AImageReader_new(width, height, AIMAGE_FORMAT_YUV_420_888, 2, &imageReader);
    if(mstatus != AMEDIA_OK) {
		LOGE("failed to create image reader (reason: %d).\n", mstatus);
		return mstatus;
    }
    LOGI("image reader created.\n");

    mstatus = AImageReader_setImageListener(imageReader, &imageListener);
    if(mstatus != AMEDIA_OK) {
		LOGE("failed to set image listener (reason: %d).\n", mstatus);
		return mstatus;
    }

    LOGI("image listener set.\n");

	AImageReader_getWindow(imageReader, &nativeWindow);
    ANativeWindow_acquire(nativeWindow);

    ACameraOutputTarget_create(nativeWindow, &cameraOutputTarget);
    ACaptureRequest_addTarget(captureRequest, cameraOutputTarget);

    ACaptureSessionOutput_create(nativeWindow, &captureSessionOutput);
	ACaptureSessionOutputContainer_add(captureSessionOutputContainer, captureSessionOutput);

    status = ACameraDevice_createCaptureSession(cameraDevice, captureSessionOutputContainer, &captureSessionStateCallbacks, &cameraCaptureSession);
    if(status != ACAMERA_OK) {
		LOGE("failed to create capture session (reason: %d).\n", status);
		return status;
    }
    LOGI("capture session created.\n");

    ACameraManager_deleteCameraIdList(cameraIdList);
    ACameraManager_delete(cameraManager);
    LOGI("camera manager deleted.\n");

    return ACAMERA_OK;
}

int captureCamera() {
    LOGI("capture camera.\n");
    camera_status_t status = ACameraCaptureSession_capture(cameraCaptureSession, NULL, 1, &captureRequest, NULL);
    if(status != ACAMERA_OK) {
		LOGE("failed to capture image (reason: %d).\n", status);
    }

    return status;
}

int closeCamera() {
    camera_status_t status = ACAMERA_OK;

    if(captureRequest != NULL) {
        ACaptureRequest_free(captureRequest);
        captureRequest = NULL;
    }

    if(cameraOutputTarget != NULL) {
        ACameraOutputTarget_free(cameraOutputTarget);
        cameraOutputTarget = NULL;
    }

    if(cameraDevice != NULL) {
        status = ACameraDevice_close(cameraDevice);

		if(status != ACAMERA_OK) {
			LOGE("failed to close camera device.\n");
			return status;
		}

		cameraDevice = NULL;
    }

    if(captureSessionOutput != NULL) {
        ACaptureSessionOutput_free(captureSessionOutput);
        captureSessionOutput = NULL;
    }

    if(captureSessionOutputContainer != NULL) {
        ACaptureSessionOutputContainer_free(captureSessionOutputContainer);
        captureSessionOutputContainer = NULL;
    }

    if(imageReader != NULL) {
		AImageReader_delete(imageReader);
		imageReader = NULL;
    }

    if(image != NULL) {
		AImage_delete(image);
		image = NULL;
	}

    LOGI("camera closed.\n");
    return ACAMERA_OK;
}

int openCamera(int index, int width, int height);
int captureCamera();
int closeCamera();

#cgo android CFLAGS: -D__ANDROID_API__=24
#cgo android LDFLAGS: -lcamera2ndk -lmediandk -llog -landroid
*/
import "C"
import (
	"context"
	"fmt"
	"image"
	"unsafe"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
)

var Model = resource.NewModel("viam", "android", "droid-cam")

type Config struct {
	Width            int                                `json:"width"`
	Height           int                                `json:"height"`
	Index            int                                `json:"index"`
	Rotate           int                                `json:"rotate"`
	Timestamp        bool                               `json:"timestamp"`
	IntrinsicParams  *transform.PinholeCameraIntrinsics `json:"intrinsic_parameters,omitempty"`
	DistortionParams *transform.BrownConrady            `json:"distortion_parameters,omitempty"`
}

func (conf *Config) Validate(path string) ([]string, error) {
	if conf.Width <= 0 {
		return nil, fmt.Errorf("width must be greater than 0")
	}
	if conf.Height <= 0 {
		return nil, fmt.Errorf("height must be greater than 0")
	}
	if conf.Index < 0 {
		return nil, fmt.Errorf("index must be greater than or equal to 0")
	}
	if conf.Rotate < 0 {
		return nil, fmt.Errorf("rotate must be greater than or equal to 0")
	}
	return nil, nil
}

type Options struct {
	Index     int
	Rotate    int
	Width     float64
	Height    float64
	Timestamp bool
}

type DroidCamera struct {
	opts       Options
	img        *image.YCbCr
	logger     logging.Logger
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	gostream.VideoReader
	started bool
}

func (c *DroidCamera) NextImage() (img image.Image, err error) {
	c.logger.Infof("### camera: hit next image")

	c.logger.Infof("### camera: camera is opened, capturing camera")
	ret := C.captureCamera()
	if bool(int(ret) != 0) {
		c.logger.Infof("### camera: failed to capture camera")
		err = fmt.Errorf("camera: can not grab frame: error %d", int(ret))
		return
	}

	C.pthread_mutex_lock(&C.globalImage.mutex)
	defer C.pthread_mutex_unlock(&C.globalImage.mutex)

	if C.globalImage.image == nil {
		c.logger.Infof("### camera: image is nil")
		err = fmt.Errorf("camera: can not retrieve frame")
		return nil, err
	}

	var yStride C.int
	var yLen, cbLen, crLen C.int
	var yPtr, cbPtr, crPtr *C.uint8_t

	C.AImage_getPlaneRowStride(C.globalImage.image, 0, &yStride)
	C.AImage_getPlaneData(C.globalImage.image, 0, &yPtr, &yLen)
	C.AImage_getPlaneData(C.globalImage.image, 1, &cbPtr, &cbLen)
	C.AImage_getPlaneData(C.globalImage.image, 2, &crPtr, &crLen)

	c.img.YStride = int(yStride)
	c.img.CStride = int(yStride) / 2

	c.img.Y = C.GoBytes(unsafe.Pointer(yPtr), yLen)
	c.img.Cb = C.GoBytes(unsafe.Pointer(cbPtr), cbLen)
	c.img.Cr = C.GoBytes(unsafe.Pointer(crPtr), crLen)

	img = c.img

	return img, nil
}

// GetProperty returns the specified camera property.
func (c *DroidCamera) GetProperty(id int) float64 {
	return 0
}

// SetProperty sets a camera property.
func (c *DroidCamera) SetProperty(id int, value float64) {
}

// Close closes camera.
func (c *DroidCamera) Close(ctx context.Context) (err error) {
	c.logger.Infof("### camera: hit close")
	ret := C.closeCamera()
	if bool(int(ret) != 0) {
		err = fmt.Errorf("camera: can not close camera %d: error %d", c.opts.Index, int(ret))
		return err
	}

	return
}

func Register() {
	resource.RegisterComponent(
		camera.API,
		Model,
		resource.Registration[camera.Camera, *Config]{
			Constructor: func(
				ctx context.Context,
				_ resource.Dependencies,
				conf resource.Config,
				logger logging.Logger,
			) (camera.Camera, error) {
				newConf, err := resource.NativeConfig[*Config](conf)
				if err != nil {
					return nil, err
				}
				return New(ctx, conf.ResourceName(), newConf, logger)
			},
		})
}

// New returns new Camera for given camera index.
func New(ctx context.Context, name resource.Name, conf *Config, logger logging.Logger) (camera.Camera, error) {
	droidCam := &DroidCamera{
		opts: Options{
			Index:     conf.Index,
			Rotate:    conf.Rotate,
			Width:     float64(conf.Width),
			Height:    float64(conf.Height),
			Timestamp: conf.Timestamp,
		},
		logger: logger,
		img:    image.NewYCbCr(image.Rect(0, 0, int(conf.Width), int(conf.Height)), image.YCbCrSubsampleRatio420),
	}

	ret := C.openCamera(C.int(conf.Index), C.int(conf.Width), C.int(conf.Height))
	if bool(int(ret) != 0) {
		return nil, fmt.Errorf("camera: can not open camera %d: error %d", conf.Index, int(ret))
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	droidCam.cancelCtx = cancelCtx
	droidCam.cancelFunc = cancel

	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
		img, err := droidCam.NextImage()
		if err != nil {
			return nil, nil, err
		}
		return img, func() {}, nil
	})
	droidCam.VideoReader = reader

	cameraModel := camera.NewPinholeModelWithBrownConradyDistortion(conf.IntrinsicParams, conf.DistortionParams)
	src, err := camera.NewVideoSourceFromReader(ctx, droidCam, &cameraModel, camera.ColorStream)
	if err != nil {
		return nil, err
	}

	return camera.FromVideoSource(name, src, logger), nil
}

func getTestImage() image.Image {
	return image.NewRGBA(image.Rect(0, 0, 640, 480))
}
