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

// BuildLifecycleWorkflow triggers a make build and make publish
func BuildLifecycleWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info("We have a new release to handle, and we will from now to the end", "StartTime", workflow.Now(ctx))

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
		},
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx1, AddRowToSmartsheet).Get(ctx, nil)
build:
	err := workflow.ExecuteActivity(ctx1, SetDetailsInSmartsheet).Get(ctx, nil)
	err := workflow.ExecuteActivity(ctx1, Build).Get(ctx, nil)
	err := workflow.ExecuteActivity(ctx1, Test).Get(ctx, nil)
	err := workflow.ExecuteActivity(ctx1, UploadBuildResultsToSmartsheet).Get(ctx, nil)
	err := workflow.ExecuteActivity(ctx1, DeclareBuildReadySmartsheet).Get(ctx, nil)

	if tag {
		for !all_signoffs {
			wait_for_signoffs
		}
		if tag_changed {
			err := workflow.ExecuteActivity(ctx1, MarkRowTamperedSmartsheet).Get(ctx, nil)
			goto end_of_life

		}
		err := workflow.ExecuteActivity(ctx1, DeployToEnv, "prod").Get(ctx, nil)
		WaitForEndOfLife
	}
	err := workflow.ExecuteActivity(ctx1, DeployToEnv, "branch_name").Get(ctx, nil)

	if signal_branch_changed {
		goto build
	}
	WaitForEndOfLife

end_of_life:
	Wait6Months
	err := workflow.ExecuteActivity(ctx1, DeleteRowFromSmartsheet).Get(ctx, nil)

	return nil
}
