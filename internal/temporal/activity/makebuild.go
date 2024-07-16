package activity

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"go.temporal.io/sdk/activity"
)

type BuildDetails struct {
	AuthKey   string
	ClonePath string
}

type BuildResponse struct {
}

// MakeBuild takes a git repo and an upload location, does the build, and then is done.
func MakeBuild(ctx context.Context, _ BuildDetails) (BuildResponse, error) {
	activity.GetLogger(ctx).Info("Executing a make build")

	path, err := exec.LookPath("make")
	if err != nil {
		newErr := fmt.Errorf("Unable to get make to run a build")
		return BuildResponse{}, errors.Join(newErr, err)
	}

	cmd := exec.Command(path, "build")
	return BuildResponse{}, cmd.Run()
}
