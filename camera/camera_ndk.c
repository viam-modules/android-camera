#include "camera_ndk.h"

GlobalImage globalImage = {NULL, PTHREAD_MUTEX_INITIALIZER};
AImage *image = NULL;
AImageReader *imageReader = NULL;
ANativeWindow *nativeWindow = NULL;
ACameraDevice *cameraDevice = NULL;
ACameraManager *cameraManager = NULL;
ACameraOutputTarget *cameraOutputTarget = NULL;
ACameraCaptureSession *cameraCaptureSession = NULL;

ACaptureRequest *captureRequest = NULL;
ACaptureSessionOutput *captureSessionOutput = NULL;
ACaptureSessionOutputContainer *captureSessionOutputContainer = NULL;

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
