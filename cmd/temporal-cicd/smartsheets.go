package main

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type SmartSheetTask struct {
	Sheet       string
	Project     string
	Version     string
	AllSignoffs bool
}

// SmartSheetNotify
func SmartSheetNotify(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Injecting a row in SmartSheet")
	return nil
}

// AddRowToSmartsheet
func AddRowToSmartsheet(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Adding a row to SmartSheet")
	return nil
}

// SetDetailsInSmartsheet
func SetDetailsInSmartsheet(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Noting details in Smartsheet")
	return nil
}

// UploadBuildResultsToSmartsheet
func UploadBuildResultsToSmartsheet(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Build and Test Logs in Smartsheet")
	return nil
}

// DeclareBuildReadySmartsheet
func DeclareBuildReadySmartsheet(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Build ready to deploy, noting so in Smartsheet")
	return nil
}

// MarkRowTamperedSmartsheet
func MarkRowTamperedSmartsheet(ctx context.Context, task SmartSheetTask) error {
	activity.GetLogger(ctx).Info("Build was tampered with, mark Untrustable")
	return nil
}

// DeployToEnv
func DeployToEnv(ctx context.Context, env_name string) error {
	activity.GetLogger(ctx).Info("Deploy time")
	return nil
}

// DeleteRowFromSmartsheet
func DeleteRowFromSmartsheet(ctx context.Context, env_name string) error {
	activity.GetLogger(ctx).Info("Deploy time")
	return nil
}
