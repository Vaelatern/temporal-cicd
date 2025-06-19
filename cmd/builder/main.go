package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sethvargo/go-envconfig"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Vaelatern/temporal-cicd/internal/config"
	"github.com/Vaelatern/temporal-cicd/internal/temporal"
)

type WorkflowInput struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	BuildPattern string `json:"build-pattern"`
	ApplyPatch   string `json:"compat-patch"`
}

type WorkflowOutput struct {
	BuildOutput  string
	UploadOutput string
}

type MakeBuilder struct {
	config config.Config
}

func (m MakeBuilder) MakeBuildUpload(ctx workflow.Context, input WorkflowInput) (WorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting MakeBuildUpload", "repo", input.Repository, "ref", input.Ref)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    10 * time.Minute,
	}

	rawCtx := workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(rawCtx, "UpdateCache", input).Get(rawCtx, nil)
	if err != nil {
		logger.Error("Failed to update cache", "error", err)
		return WorkflowOutput{}, err
	}

	// Session Start
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

	var buildOutput string
	var uploadOutput string
	{

		sessionCtx = workflow.WithActivityOptions(sessionCtx, ao)

		var cloneOutput string
		err = workflow.ExecuteActivity(sessionCtx, "DownloadFromCacheActivity", input).Get(sessionCtx, &cloneOutput)
		if err != nil {
			logger.Error("Failed to fetch from cache", "error", err)
			return WorkflowOutput{}, err
		}

		err = workflow.ExecuteActivity(sessionCtx, "BuildActivity", cloneOutput).Get(sessionCtx, &buildOutput)
		if err != nil {
			logger.Error("Failed to run make build", "error", err)
			return WorkflowOutput{}, err
		}

		err = workflow.ExecuteActivity(sessionCtx, "UploadActivity", cloneOutput).Get(sessionCtx, &uploadOutput)
		if err != nil {
			logger.Error("Failed to run make upload", "error", err)
			return WorkflowOutput{}, err
		}
	}

	return WorkflowOutput{
		BuildOutput:  buildOutput,
		UploadOutput: uploadOutput,
	}, nil
}

func (m MakeBuilder) UpdateCache(ctx context.Context, args WorkflowInput) error {
	repo := args.Repository
	ref := args.Ref
	logger := activity.GetLogger(ctx)
	logger.Info("Triggering cache", "repo", repo, "ref", ref)

	tarballURL := fmt.Sprintf("%s/sync/%s/%s", m.config.CacheURL, repo, ref)

	client := &http.Client{}
	req, err := http.NewRequest("POST", tarballURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer a")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger cache: %w", err)
	}
	return resp.Body.Close()
}

func (m MakeBuilder) DownloadFromCacheActivity(ctx context.Context, args WorkflowInput) (string, error) {
	repo := args.Repository
	ref := args.Ref
	logger := activity.GetLogger(ctx)
	logger.Info("Downloading tarball from cache", "repo", repo, "ref", ref)

	tmpDir := filepath.Join("/dev/shm", uuid.New().String())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tarballURL := fmt.Sprintf("%s/download/%s/%s", m.config.CacheURL, repo, ref)

	client := &http.Client{}
	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer a")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	cmd := exec.CommandContext(ctx, "tar", "-xz", "-C", tmpDir)
	cmd.Stdin = resp.Body
	logger.Info("Starting process")
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("Failed to start tar command: %v\n", err)
	}

	cmd.Wait()
	logger.Info("Tarball extracted", "path", tmpDir)
	return tmpDir, nil
}

func (m MakeBuilder) BuildActivity(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "build")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (m MakeBuilder) UploadActivity(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "upload")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
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
