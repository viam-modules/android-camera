// Package androidcamera.
package androidcamera

// #cgo android LDFLAGS: -lcamera2ndk -lmediandk -llog -landroid
// #include "camera_ndk.h"
import "C"
import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
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

// Validate validates the configuration.
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
	if conf.Rotate%90 != 0 {
		return nil, fmt.Errorf("rotate must be a multiple of 90")
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
	logger     logging.Logger
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	gostream.VideoReader
	started bool
}

// NextImage returns the next image from the camera.
func (c *DroidCamera) NextImage() (img image.Image, err error) {
	ret := C.captureCamera()
	if bool(int(ret) != 0) {
		err = fmt.Errorf("camera: can not grab frame: error %d", int(ret))
		return nil, err
	}

	C.pthread_mutex_lock(&C.globalImage.mutex)
	defer C.pthread_mutex_unlock(&C.globalImage.mutex)

	if C.globalImage.image == nil {
		err = fmt.Errorf("camera: image is nil")
		return nil, err
	}

	var jpegLen C.int
	var jpegPtr *C.uint8_t

	C.AImage_getPlaneData(C.globalImage.image, 0, &jpegPtr, &jpegLen)
	jpegData := C.GoBytes(unsafe.Pointer(jpegPtr), jpegLen)

	img, err = jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil, fmt.Errorf("camera: failed to decode JPEG: %v", err)
	}

	img = rotateImage(img, c.opts.Rotate)

	return img, nil
}

// Close closes camera.
func (c *DroidCamera) Close(ctx context.Context) (err error) {
	ret := C.closeCamera()
	if bool(int(ret) != 0) {
		err = fmt.Errorf("camera: can not close camera %d: error %d", c.opts.Index, int(ret))
		return err
	}

	return
}

// Register sets up the camera component.
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

// rotateImage rotates the image by 0, 90, 180, or 270 degrees clockwise.
// Only these four angles are supported due to the simplicity of the algorithm.
func rotateImage(img image.Image, rotate int) image.Image {
	bounds := img.Bounds()
	var rotated *image.NRGBA

	switch rotate % 360 {
	case 90:
		rotated = image.NewNRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rotated.Set(bounds.Dy()-y-1, x, img.At(x, y))
			}
		}
	case 180:
		rotated = image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rotated.Set(bounds.Dx()-x-1, bounds.Dy()-y-1, img.At(x, y))
			}
		}
	case 270:
		rotated = image.NewNRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rotated.Set(y, bounds.Dx()-x-1, img.At(x, y))
			}
		}
	default: // includes 0 and multiples of 360
		rotated = image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rotated.Set(x, y, img.At(x, y))
			}
		}
	}

	return rotated
}

// getTestImage returns a test image.
func getTestImage() image.Image {
	return image.NewRGBA(image.Rect(0, 0, 640, 480))
}
