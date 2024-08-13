package activity

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/leonklingele/securetemp"
	"go.temporal.io/sdk/activity"
)

type BuildReference struct {
	Full      string
	Name      string
	Hash      string
	Permanent bool
}

type BuildDetails struct {
	CloneUser string
	CloneKey  string
	ClonePath string
	Reference BuildReference
}

type BuildResponse struct {
}

// MakeBuild takes a git repo and an upload location, does the build, and then is done.
func MakeBuild(ctx context.Context, bd BuildDetails) (BuildResponse, error) {
	activity.GetLogger(ctx).Info("Executing a make build")

	workdir, cleanupCallback, err := securetemp.TempDir(2 * securetemp.SizeGB)
	defer cleanupCallback()

	cloneAuth, err := ssh.NewPublicKeys(bd.CloneUser, []byte(bd.CloneKey), "")
	if err != nil {
		newErr := fmt.Errorf("Public Keys unusable")
		return BuildResponse{}, errors.Join(newErr, err)
	}

	cloneOps := git.CloneOptions{
		Auth:          cloneAuth,
		URL:           bd.ClonePath,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(bd.Reference.Full),
		Depth:         1,
		Tags:          git.NoTags,
	}

	_, err = git.PlainClone(workdir, false, &cloneOps)
	if err != nil {
		newErr := fmt.Errorf("Unable to clone")
		return BuildResponse{}, errors.Join(newErr, err)
	}

	path, err := exec.LookPath("make")
	if err != nil {
		newErr := fmt.Errorf("Unable to get make to run a build")
		return BuildResponse{}, errors.Join(newErr, err)
	}

	cmd := exec.CommandContext(ctx, path, "build")
	cmd.Dir = workdir

	err = cmd.Run()

	if err != nil {
		newErr := fmt.Errorf("Build step failed")
		return BuildResponse{}, errors.Join(newErr, err)
	}

	cmd = exec.CommandContext(ctx, path, "publish")
	cmd.Dir = workdir

	err = cmd.Run()

	return BuildResponse{}, err
}
