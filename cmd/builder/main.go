package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sethvargo/go-envconfig"
	"go.temporal.io/sdk/worker"

	"github.com/Vaelatern/temporal-cicd/internal/config"
	"github.com/Vaelatern/temporal-cicd/internal/temporal"
)

type WorkflowInput struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	BuildPattern string `json:"build-pattern"`
	ApplyPatch   string `json:"compat-patch"`
}

func main() {
	var conf config.Config
	if err := envconfig.Process(context.Background(), &conf); err != nil {
		log.Fatal(err)
	}

	c, err := temporal.EasyClient(temporal.Logger())
	if err != nil {
		fmt.Printf("Failed to create Temporal client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, "basic-builder", worker.Options{
		EnableSessionWorker: true,
	})

	m := MakeBuilder{
		config: conf,
	}

	w.RegisterWorkflow(m.MakeBuildUpload)
	w.RegisterActivity(m.UpdateCache)
	w.RegisterActivity(m.DownloadFromCacheActivity)
	w.RegisterActivity(m.BuildActivity)
	w.RegisterActivity(m.UploadActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		fmt.Printf("Failed to start worker: %v\n", err)
		os.Exit(1)
	}
}
