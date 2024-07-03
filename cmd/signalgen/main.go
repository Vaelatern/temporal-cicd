package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type ProcessedRequest struct {
	Project           string
	Version           string
	Signoffs          string
	TargetEnvironment string
}

func main() {
	fmt.Println("vim-go")
	reqs := make(chan ProcessedRequest, 10)

	http.HandleFunc("/ss", SSListener(reqs))
	http.HandleFunc("/git", GitListener(reqs))
	go DeploySignal(reqs)
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
func DeploySignal(reqs <-chan ProcessedRequest) {
	for {
		select {
		case req := <-reqs:
			fmt.Println(req)
		}
		signal := SmartSheetTask{}
		err = temporalClient.SignalWorkflow(context.Background(), "your-workflow-id", runID, "message-from-smartsheet", signal)
		if err != nil {
			log.Fatalln("Error sending the Signal", err)
			return
		}
	}
}
