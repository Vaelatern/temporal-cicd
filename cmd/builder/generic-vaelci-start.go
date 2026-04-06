package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Vaelatern/temporal-cicd/internal/config"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// VaelCiConfig represents the structure of .vaelci.json
type VaelCiConfig struct {
	BuildPattern string `json:"build-pattern"`
}

// GenericBuilder handles the generic start workflow that routes to the correct build workflow
type GenericBuilder struct {
	config config.Config
}

// DetermineSpecificBuildFlow downloads the tarball from cache, extracts just .vaelci.json,
// parses it as JSON, and returns the VaelCiConfig.
func (g GenericBuilder) DetermineSpecificBuildFlow(ctx context.Context, input WorkflowInput) (VaelCiConfig, error) {
	repo := input.Repository
	ref := input.Ref
	logger := activity.GetLogger(ctx)
	logger.Info("Determining build flow from .vaelci.json", "repo", repo, "ref", ref)

	// Download tarball from cache
	tarballURL := fmt.Sprintf("%s/download/%s/%s", g.config.CacheURL, repo, ref)

	client := &http.Client{}
	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return VaelCiConfig{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer a")
	resp, err := client.Do(req)
	if err != nil {
		return VaelCiConfig{}, fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	// Read the tarball and look for .vaelci.json
	tr := tar.NewReader(resp.Body)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return VaelCiConfig{}, fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for .vaelci.json at the root of the repository
		if header.Name == ".vaelci.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return VaelCiConfig{}, fmt.Errorf("failed to read .vaelci.json: %w", err)
			}

			var config VaelCiConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return VaelCiConfig{}, fmt.Errorf("failed to parse .vaelci.json: %w", err)
			}

			logger.Info("Successfully read .vaelci.json", "build-pattern", config.BuildPattern)
			return config, nil
		}
	}

	return VaelCiConfig{}, fmt.Errorf(".vaelci.json not found in tarball")
}

// GenericVaelCiCdStart is a workflow that fetches .vaelci.json from the repository,
// determines the build pattern, and starts the child workflow.
// It exits once the child workflow has started.
func (g GenericBuilder) GenericVaelCiCdStart(ctx workflow.Context, input WorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting GenericVaelCiCdStart", "repo", input.Repository, "ref", input.Ref)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    10 * time.Minute,
	}

	rawCtx := workflow.WithActivityOptions(ctx, ao)

	// Determine build flow by reading .vaelci.json from tarball
	var vaelciConfig VaelCiConfig
	err := workflow.ExecuteActivity(rawCtx, "DetermineSpecificBuildFlow", input).Get(rawCtx, &vaelciConfig)
	if err != nil {
		logger.Error("Failed to determine build flow", "error", err)
		return err
	}

	logger.Info("Determined build pattern", "build-pattern", vaelciConfig.BuildPattern)

	// Start child workflow with the build pattern from .vaelci.json
	input.BuildPattern = vaelciConfig.BuildPattern
	childErr := workflow.ExecuteChildWorkflow(ctx, vaelciConfig.BuildPattern, input).Get(ctx, nil)
	if childErr != nil {
		logger.Error("Child workflow failed", "error", childErr)
		return childErr
	}

	logger.Info("Child workflow completed successfully")
	return nil
}
