package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"go.temporal.io/sdk/activity"
)

// MakeBuild takes a git repo and an upload location, does the build, and then is done.
func MakeBuild(ctx context.Context, gitRepo string) error {
	activity.GetLogger(ctx).Info("Executing a make build", "gitRepo", gitRepo)

	path, err := exec.LookPath("make")
	if err != nil {
		newErr := fmt.Errorf("Unable to get make to run a build")
		return errors.Join(newErr, err)
	}

	cmd := exec.Command(path, "build")
	return cmd.Run()
}
