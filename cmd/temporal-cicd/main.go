package main

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"
)

func main() {
	logger := log.NewStructuredLogger(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})))
	clientOptions := client.Options{
		Logger: logger,
	}
	temporalClient, err := client.Dial(clientOptions)

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
