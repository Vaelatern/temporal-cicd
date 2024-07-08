package temporal

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
)

func Logger() log.Logger {
	return log.NewStructuredLogger(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})))
}

func EasyClient(logger log.Logger) (client.Client, error) {
	temporalHost, useDefault := os.LookupEnv("TEMPORAL_ADDRESS")
	if useDefault {
		temporalHost = "localhost:7233"
	}
	clientOptions := client.Options{
		HostPort:    temporalHost,
		Credentials: nil,
		Logger:      logger,
	}
	temporalClient, err := client.Dial(clientOptions)
	return temporalClient, err
}
