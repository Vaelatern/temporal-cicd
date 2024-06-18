package main

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type SmartSheetTask struct {
	Sheet   string
	Project string
	Version string
}

// SmartSheetNotify takes a git repo and an upload location, does the build, and then is done.
func SmartSheetNotify(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Injecting a row in SmartSheet")

	return nil
}
