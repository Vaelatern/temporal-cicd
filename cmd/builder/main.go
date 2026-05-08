package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	temporal_envconfig "go.temporal.io/sdk/contrib/envconfig"
	"go.temporal.io/sdk/contrib/sysinfo"
	temporal_log "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"

	"github.com/Vaelatern/temporal-cicd/internal/config"
)

type WorkflowInput struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	BuildPattern string `json:"build-pattern"`
	ApplyPatch   string `json:"compat-patch"`
}

func logger() temporal_log.Logger {
	return temporal_log.NewStructuredLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))
}

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	opts := temporal_envconfig.MustLoadDefaultClientOptions()
	opts.Logger = logger()
	c, err := client.Dial(opts)
	if err != nil {
		fmt.Printf("Failed to create Temporal client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, "basic-builder", worker.Options{
		EnableSessionWorker: true,
		SysInfoProvider:     sysinfo.SysInfoProvider(),
	})

	m := MakeBuilder{
		config: *conf,
	}

	g := GenericBuilder{
		config: *conf,
	}

	w.RegisterWorkflow(m.MakeBuildUpload)
	w.RegisterActivity(m.UpdateCache)
	w.RegisterActivity(m.DownloadFromCacheActivity)
	w.RegisterActivity(m.BuildActivity)
	w.RegisterActivity(m.UploadActivity)
	w.RegisterWorkflow(g.GenericVaelCiCdStart)
	w.RegisterActivity(g.DetermineSpecificBuildFlow)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		fmt.Printf("Failed to start worker: %v\n", err)
		os.Exit(1)
	}
}
