package main

import (
	"github.com/Vaelatern/temporal-cicd/internal/config"
	"go.temporal.io/sdk/workflow"
)

type WorkflowInput struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	BuildPattern string `json:"build-pattern"`
	ApplyPatch   string `json:"compat-patch"`
}

// I bet the right approach here is a file mapping on the server
// Let us match with regular expressions to decide what deploy
// phases get used.
// Future enhancement.
// Especially since I like bytecode interpreters, and then I might
// have the configuration be arbitrarily complex steps like conditionals
// and looping -- or maybe just declarative regex matching against the build
// args when deciding things (and of course variables can be used).

type Deployer struct {
	config config.Config
}

type DeploymentInput struct {
}

type DeploymentOutput struct {
}

func (d Deployer) Deployment(ctx workflow.Context, input WorkflowInput) (DeploymentOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Deployment", "repo", input.Repository, "ref", input.Ref)
	return DeploymentOutput{}, nil
}
