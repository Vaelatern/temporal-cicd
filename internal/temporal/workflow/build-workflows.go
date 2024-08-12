package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Vaelatern/temporal-cicd/internal/temporal/activity"
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

	err := workflow.ExecuteActivity(ctx1, activity.PodmanBuild, "https://github.com/Vaelatern/http-pdf-generator").Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("schedule workflow failed.", "Error", err)
		return err
	}
	return nil
}

// MakeBuildWorkflow triggers a make build and make publish
func MakeBuildWorkflow(ctx workflow.Context, build_coords activity.BuildDetails) error {
	workflow.GetLogger(ctx).Info("Starting make build", "StartTime", workflow.Now(ctx))

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
		},
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)

	br := activity.BuildResponse{}

	err := workflow.ExecuteActivity(ctx1, activity.MakeBuild, build_coords).Get(ctx, &br)
	if err != nil {
		workflow.GetLogger(ctx).Error("schedule workflow failed.", "Error", err)
		return err
	}
	return nil
}
