package main

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PodmanBuildWorkflow triggers a podman build and publish
func PodmanBuildWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info("Starting podman build", "StartTime", workflow.Now(ctx))

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
		},
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx1, PodmanBuild, "https://github.com/Vaelatern/http-pdf-generator").Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("schedule workflow failed.", "Error", err)
		return err
	}
	return nil
}

// MakeBuildWorkflow triggers a make build and make publish
func MakeBuildWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info("Starting podman build", "StartTime", workflow.Now(ctx))

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
		},
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx1, MakeBuild, "https://github.com/Vaelatern/http-pdf-generator").Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("schedule workflow failed.", "Error", err)
		return err
	}
	return nil
}
