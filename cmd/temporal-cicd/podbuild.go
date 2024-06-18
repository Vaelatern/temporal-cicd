package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"go.temporal.io/sdk/activity"
)

// PodmanBuild takes a git repo and an upload location, does the build, and then is done.
func PodmanBuild(ctx context.Context, gitRepo string) error {
	activity.GetLogger(ctx).Info("Executing a podman build", "gitRepo", gitRepo)

	path, err := exec.LookPath("podman")
	if err != nil {
		newErr := fmt.Errorf("Unable to get podman to run a build")
		return errors.Join(newErr, err)
	}

	cmd := exec.Command(path, "build", ".")
	return cmd.Run()
}
