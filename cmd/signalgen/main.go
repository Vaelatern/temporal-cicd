package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Vaelatern/temporal-cicd/internal/event"
	"github.com/Vaelatern/temporal-cicd/internal/temporal"
)

type ProcessedRequest struct {
	Project           string
	Version           string
	Signoffs          string
	TargetEnvironment string
}

func main() {
	reqs := make(chan ProcessedRequest, 10)

	temporalClient, err := temporal.EasyClient(temporal.Logger())
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/ss", SSListener(reqs))
	http.HandleFunc("/git", GitListener(reqs))
	go DeploySignal(temporalClient, reqs)
	log.Fatal(http.ListenAndServe("", nil))
}

// SSListener is a handler for receiving and parsing webhooks from SmartSheets
func SSListener(reqs chan<- ProcessedRequest) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reqs <- ProcessedRequest{}
	}
}

// GitListener is a handler for receiving and parsing webhooks from our git source
func GitListener(reqs chan<- ProcessedRequest) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reqs <- ProcessedRequest{}
	}
}

// DeploySignal signals an appropriate workflow to promote to an environment
func DeploySignal(temporalClient client.Client, reqs <-chan ProcessedRequest) {
	for {
		select {
		case req := <-reqs:
			fmt.Println(req)
		}
		signal := event.SmartSheetTask{}
		err := temporalClient.SignalWorkflow(context.Background(), "your-workflow-id", "", "message-from-smartsheet", signal)
		if err != nil {
			log.Fatalln("Error sending the Signal", err)
			return
		}
	}
}
