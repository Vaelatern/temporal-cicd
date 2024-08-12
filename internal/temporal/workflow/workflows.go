package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Vaelatern/temporal-cicd/internal/event"
	"github.com/Vaelatern/temporal-cicd/internal/temporal/activity"
)

// SmartSheetManagedBuildLifecycleWorkflow triggers a make build and make publish
func SmartSheetManagedBuildLifecycleWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info("We have a new release to handle, and we will from now to the end", "StartTime", workflow.Now(ctx))

	b := activity.BuildDetails{}
	smartsheetSignal := workflow.GetSignalChannel(ctx, "message-from-smartsheet")
	gitSignalChan := workflow.GetSignalChannel(ctx, "message-from-git")

	var signal event.SmartSheetTask
	end_of_life_timeout := 6 * 31 * 24 * time.Hour

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
		},
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx1, activity.AddRowToSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}
build:
	err = workflow.ExecuteActivity(ctx1, activity.SetDetailsInSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}
	err = workflow.ExecuteChildWorkflow(ctx, MakeBuildWorkflow, b).Get(ctx, nil)
	if err != nil {
		return err
	}
	err = workflow.ExecuteActivity(ctx1, activity.UploadBuildResultsToSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}
	err = workflow.ExecuteActivity(ctx1, activity.DeclareBuildReadySmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}

	if b.Reference.Permanent {
		for {
			smartsheetSignal.Receive(ctx, &signal)
			if signal.AllSignoffs {
				break
			}
		}
		err := workflow.ExecuteActivity(ctx1, activity.DeployToEnv, activity.DeployEnv{Name: "prod"}).Get(ctx, nil)
		if err != nil {
			return err
		}
	} else {
		err = workflow.ExecuteActivity(ctx1, activity.DeployToEnv, activity.DeployEnv{Name: b.Reference.Name}).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

end_of_life:
	received, _ := gitSignalChan.ReceiveWithTimeout(ctx, end_of_life_timeout, &signal)
	if received && b.Reference.Permanent {
		err := workflow.ExecuteActivity(ctx1, activity.MarkRowTamperedSmartsheet).Get(ctx, nil)
		if err != nil {
			return err
		}
		goto end_of_life
	} else if received {
		goto build
	}

	err = workflow.ExecuteActivity(ctx1, activity.DeleteRowFromSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
