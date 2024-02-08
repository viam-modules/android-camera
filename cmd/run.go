package main

import (
	"context"

	androidcamera "github.com/seanavery/viam-droid-cam/camera"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
)

func main() {
	androidcamera.Register()

	ctx := context.Background()
	logger := logging.NewDebugLogger("droidcamera")

	myMod, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		logger.Errorf("Failed to create module: %v", err)
		return
	}

	err = myMod.AddModelFromRegistry(ctx, camera.API, androidcamera.Model)
	if err != nil {
		logger.Errorf("Failed to add model: %v", err)
		return
	}

	err = myMod.Start(ctx)
	if err != nil {
		logger.Errorf("Failed to start module: %v", err)
		return
	}
	defer myMod.Close(ctx)

	<-ctx.Done()
}
