package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type WorkflowInput struct {
	RepoName string
	Ref      string
}

type WorkflowOutput struct {
	BuildOutput  string
	UploadOutput string
}

func GitBuildUploadWorkflow(ctx workflow.Context, input WorkflowInput) (WorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting GitBuildUploadWorkflow", "repo", input.RepoName, "ref", input.Ref)

	sessionOptions := &workflow.SessionOptions{
		ExecutionTimeout: 30 * time.Minute,
		HeartbeatTimeout: 1 * time.Minute,
	}
	sessionCtx, err := workflow.CreateSession(ctx, sessionOptions)
	if err != nil {
		logger.Error("Failed to create session", "error", err)
		return WorkflowOutput{}, err
	}
	defer workflow.CompleteSession(sessionCtx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    1 * time.Minute,
	}
	sessionCtx = workflow.WithActivityOptions(sessionCtx, ao)

	var cloneOutput string
	err = workflow.ExecuteActivity(sessionCtx, DownloadFromCacheActivity, input.RepoName, input.Ref).Get(sessionCtx, &cloneOutput)
	if err != nil {
		logger.Error("Failed to fetch from cache", "error", err)
		return WorkflowOutput{}, err
	}

	var buildOutput string
	err = workflow.ExecuteActivity(sessionCtx, BuildActivity, cloneOutput).Get(sessionCtx, &buildOutput)
	if err != nil {
		logger.Error("Failed to run make build", "error", err)
		return WorkflowOutput{}, err
	}

	var uploadOutput string
	err = workflow.ExecuteActivity(sessionCtx, UploadActivity, cloneOutput).Get(sessionCtx, &uploadOutput)
	if err != nil {
		logger.Error("Failed to run make upload", "error", err)
		return WorkflowOutput{}, err
	}

	return WorkflowOutput{
		BuildOutput:  buildOutput,
		UploadOutput: uploadOutput,
	}, nil
}

func DownloadFromCacheActivity(ctx context.Context, repo, ref string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Downloading tarball from cache", "repo", repo, "ref", ref)

	tmpDir := filepath.Join("/dev/shm", uuid.New().String())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tarballURL := fmt.Sprintf("http://cache:8080/download/%s/%s.tar.gz", repo, ref)
	resp, err := http.Get(tarballURL)
	if err != nil {
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", err
	}
	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		path := filepath.Join(tmpDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			os.MkdirAll(path, 0755)
		} else {
			os.MkdirAll(filepath.Dir(path), 0755)
			f, err := os.Create(path)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
		}
	}

	logger.Info("Tarball extracted", "path", tmpDir)
	return tmpDir, nil
}

func BuildActivity(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "build")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func UploadActivity(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "upload")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		fmt.Printf("Failed to create Temporal client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, "git-build-upload-task-queue", worker.Options{
		EnableSessionWorker: true,
	})

	w.RegisterWorkflow(GitBuildUploadWorkflow)
	w.RegisterActivity(DownloadFromCacheActivity)
	w.RegisterActivity(BuildActivity)
	w.RegisterActivity(UploadActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		fmt.Printf("Failed to start worker: %v\n", err)
		os.Exit(1)
	}
}
