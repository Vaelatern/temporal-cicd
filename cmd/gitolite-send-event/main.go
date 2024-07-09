package main

import (
	"context"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/Vaelatern/temporal-cicd/internal/temporal"
)

func main() {
	ctx := context.Background()

	temporalClient, err := temporal.EasyClient(temporal.Logger())
	if err != nil {
		log.Fatal(err)
	}

	taskQueue := "cicd"
	project := "temporal-cicd"
	commitName := "master"

	firstWorkflowID := project + "/" + commitName
	firstWorkflowOptions := client.StartWorkflowOptions{
		ID:                       firstWorkflowID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: 5 * time.Minute,
	}
	_, err = temporalClient.ExecuteWorkflow(ctx, firstWorkflowOptions, "SampleChangingWorkflow")
	if err != nil {
		log.Fatalln("Unable to start workflow", err)
	}
}
