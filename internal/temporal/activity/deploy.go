package activity

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type DeployEnv struct {
	Name string
}

type DeployEnvResponse struct{}

// DeployToEnv
func DeployToEnv(ctx context.Context, deploy DeployEnv) (DeployEnvResponse, error) {
	activity.GetLogger(ctx).Info("Deploy time")
	return DeployEnvResponse{}, nil
}
