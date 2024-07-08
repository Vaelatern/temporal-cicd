package main

import (
	"go.temporal.io/sdk/worker"

	"github.com/Vaelatern/temporal-cicd/internal/temporal"
)

func main() {
	logger := temporal.Logger()

	temporalClient, err := temporal.EasyClient(logger)

	if err != nil {
		logger.Error("Unable to create client", err)
	}

	defer temporalClient.Close()

	w := worker.New(temporalClient, "cicd", worker.Options{})

	w.RegisterActivity(SmartSheetNotify)

	w.RegisterWorkflow(PodmanBuildWorkflow)
	w.RegisterActivity(PodmanBuild)

	w.RegisterWorkflow(MakeBuildWorkflow)
	w.RegisterActivity(MakeBuild)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("Unable to start worker", err)
	}
}
