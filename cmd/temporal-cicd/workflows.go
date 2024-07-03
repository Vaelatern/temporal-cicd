package main

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type BuildDetails struct {
	name        string
	hash        string
	permanent   bool
	updateindex int
}

// BuildLifecycleWorkflow triggers a make build and make publish
func BuildLifecycleWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info("We have a new release to handle, and we will from now to the end", "StartTime", workflow.Now(ctx))

	b := BuildDetails{}
	signalChan := workflow.GetSignalChannel(ctx, "message-from-smartsheet")

	var signal SmartSheetTask
	end_of_life_timeout := 6 * 31 * 24 * time.Hour

	var tag bool = false

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
		},
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx1, AddRowToSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}
build:
	err = workflow.ExecuteActivity(ctx1, SetDetailsInSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}
	var build_result error
	err = workflow.ExecuteChildWorkflow(ctx, MakeBuildWorkflow).Get(ctx, &build_result)
	if err != nil {
		return err
	}
	err = workflow.ExecuteActivity(ctx1, UploadBuildResultsToSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}
	err = workflow.ExecuteActivity(ctx1, DeclareBuildReadySmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}

	if tag {
		for {
			signalChan.Receive(ctx, &signal)
			if signal.AllSignoffs {
				break
			}
		}
		err := workflow.ExecuteActivity(ctx1, DeployToEnv, "prod").Get(ctx, nil)
		if err != nil {
			return err
		}
	} else {
		err = workflow.ExecuteActivity(ctx1, DeployToEnv, "branch_name").Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	for {
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &signal)
		})
		selector.Select(ctx)
	}

end_of_life:
	received, _ := signalChan.ReceiveWithTimeout(ctx, end_of_life_timeout, &signal)
	if received && b.permanent {
		err := workflow.ExecuteActivity(ctx1, MarkRowTamperedSmartsheet).Get(ctx, nil)
		if err != nil {
			return err
		}
		goto end_of_life
	} else if received {
		goto build
	}

	err = workflow.ExecuteActivity(ctx1, DeleteRowFromSmartsheet).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
