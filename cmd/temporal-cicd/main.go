package main

import (
	"os"

	"go.temporal.io/sdk/worker"

	"github.com/Vaelatern/temporal-cicd/internal/temporal"
	"github.com/Vaelatern/temporal-cicd/internal/temporal/activity"
	"github.com/Vaelatern/temporal-cicd/internal/temporal/workflow"
)

func main() {
	logger := temporal.Logger()

	temporalClient, err := temporal.EasyClient(logger)

	if err != nil {
		logger.Error("Unable to create client", err)
		os.Exit(1)
	}

	defer temporalClient.Close()

	w := worker.New(temporalClient, "cicd", worker.Options{})

	w.RegisterWorkflow(workflow.PodmanBuildWorkflow)
	w.RegisterActivity(activity.PodmanBuild)

	w.RegisterWorkflow(workflow.MakeBuildWorkflow)
	w.RegisterActivity(activity.MakeBuild)

	w.RegisterWorkflow(workflow.SmartSheetManagedBuildLifecycleWorkflow)

	w.RegisterActivity(activity.SmartSheetNotify)
	w.RegisterActivity(activity.AddRowToSmartsheet)
	w.RegisterActivity(activity.SetDetailsInSmartsheet)
	w.RegisterActivity(activity.UploadBuildResultsToSmartsheet)
	w.RegisterActivity(activity.DeclareBuildReadySmartsheet)
	w.RegisterActivity(activity.MarkRowTamperedSmartsheet)
	w.RegisterActivity(activity.DeployToEnv)
	w.RegisterActivity(activity.DeleteRowFromSmartsheet)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("Unable to start worker", err)
	}
}
