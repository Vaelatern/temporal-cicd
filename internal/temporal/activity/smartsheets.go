package activity

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type SmartSheetTask struct{}
type SmartSheetTaskResponse struct{}

// SmartSheetNotify
func SmartSheetNotify(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Injecting a row in SmartSheet")
	return SmartSheetTaskResponse{}, nil
}

// AddRowToSmartsheet
func AddRowToSmartsheet(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Adding a row to SmartSheet")
	return SmartSheetTaskResponse{}, nil
}

// SetDetailsInSmartsheet
func SetDetailsInSmartsheet(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Noting details in Smartsheet")
	return SmartSheetTaskResponse{}, nil
}

// UploadBuildResultsToSmartsheet
func UploadBuildResultsToSmartsheet(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Build and Test Logs in Smartsheet")
	return SmartSheetTaskResponse{}, nil
}

// DeclareBuildReadySmartsheet
func DeclareBuildReadySmartsheet(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Build ready to deploy, noting so in Smartsheet")
	return SmartSheetTaskResponse{}, nil
}

// MarkRowTamperedSmartsheet
func MarkRowTamperedSmartsheet(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Build was tampered with, mark Untrustable")
	return SmartSheetTaskResponse{}, nil
}

// DeleteRowFromSmartsheet
func DeleteRowFromSmartsheet(ctx context.Context, task SmartSheetTask) (SmartSheetTaskResponse, error) {
	activity.GetLogger(ctx).Info("Deploy time")
	return SmartSheetTaskResponse{}, nil
}
