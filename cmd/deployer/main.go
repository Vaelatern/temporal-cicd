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

	w := worker.New(c, "deployer", worker.Options{
		EnableSessionWorker: true,
		SysInfoProvider:     sysinfo.SysInfoProvider(),
	})

	d := Deployer{
		config: *conf,
	}

	w.RegisterWorkflow(d.Deployment)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		fmt.Printf("Failed to start worker: %v\n", err)
		os.Exit(1)
	}
}
