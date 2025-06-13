#!/bin/sh
set -e

REPO_DIR="/repos"

if [ ! -d "$REPO_DIR/.git" ]; then
  git init --bare "$REPO_DIR"
fi

cd "$REPO_DIR"
git config --bool core.bare true
git config receive.denyCurrentBranch ignore
echo "Initialized bare git repository at $REPO_DIR"
