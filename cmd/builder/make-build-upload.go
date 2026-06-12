package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Vaelatern/temporal-cicd/internal/config"
	"github.com/google/uuid"
	"github.com/hashicorp/go-extract"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type MakeBuilder struct {
	config config.Config
}

type WorkflowOutput struct {
	BuildOutput  string
	UploadOutput string
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

	// Start Deployment child workflow on the "deployer" task queue.
	// Abandon the child (do not wait for completion) but keep the parent
	// operational until the child has launched successfully.
	info := workflow.GetInfo(ctx)
	childWorkflowID := info.WorkflowExecution.ID + "-deployer"
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: childWorkflowID,
		TaskQueue:  "deployer",
	})
	childFuture := workflow.ExecuteChildWorkflow(childCtx, "Deployment", input)
	// Wait only until launch (child started); then abandon it
	if err := childFuture.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		logger.Error("Failed to launch deployer child workflow", "error", err)
		return WorkflowOutput{}, err
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

	tarballURL := fmt.Sprintf("%s/sync/%s/%s", m.config.Cache.URL, url.PathEscape(repo), url.PathEscape(ref))

	client := &http.Client{}
	req, err := http.NewRequest("POST", tarballURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	m.config.AddCacheHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger cache: %w", err)
	}
	return resp.Body.Close()
}

func extractTypeFromDisposition(disposition string) string {
	if disposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return ""
	}
	filename, ok := params["filename"]
	if !ok || filename == "" {
		return ""
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	return ext
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

	tarballURL := fmt.Sprintf("%s/download/%s/%s", m.config.Cache.URL, url.PathEscape(repo), url.PathEscape(ref))

	client := &http.Client{}
	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	m.config.AddCacheHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("cache returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	cdHeader := resp.Header.Get("Content-Disposition")
	extractType := extractTypeFromDisposition(cdHeader)
	opts := []extract.ConfigOption{}
	if extractType != "" {
		logger.Info("Detected archive type from Content-Disposition filename", "type", extractType)
		opts = append(opts, extract.WithExtractType(extractType))
	} else {
		logger.Warn("No archive type in Content-Disposition, falling back to magic-byte detection", "content-disposition", cdHeader)
	}

	if err := extract.Unpack(ctx, tmpDir, resp.Body, extract.NewConfig(opts...)); err != nil {
		return "", fmt.Errorf("Failed to start tar command: %v\n", err)
	}

	logger.Info("Tarball extracted", "path", tmpDir)
	return tmpDir, nil
}

func (m MakeBuilder) BuildActivity(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "build")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("make build failed (%w):\n%s", err, string(output))
	}
	return string(output), nil
}

func (m MakeBuilder) UploadActivity(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "make", "upload")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("make upload failed (%w):\n%s", err, string(output))
	}
	return string(output), nil
}
