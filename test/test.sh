#!/bin/bash

# This script assumes:
# - GITOLITE_ADDR is set, e.g., export GITOLITE_ADDR="git@192.168.1.100:22"
# - SSH access to gitolite is configured without password prompts (keys set up).
# - Run this script in an empty directory.
# - The new repo name is "testrepo" (change if needed).
# - For simplicity, the gitolite config grants RW+ to @all (adjust for production).

set -e  # Exit on error

# Check if GITOLITE_ADDR is set
if [ -z "$GITOLITE_ADDR" ]; then
  echo "Error: GITOLITE_ADDR must be set (e.g., git@ip:port)"
  exit 1
fi

export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

# Clone gitolite-admin
git clone "ssh://$GITOLITE_ADDR/gitolite-admin.git"
cd gitolite-admin

# Add new repo to config
printf "\nrepo testrepo\n    RW+     =   @all\n" >> conf/gitolite.conf
printf "\nrepo @all\n    - lrcicd     =   @all\n" >> conf/gitolite.conf

# Commit and push config changes
git add conf/gitolite.conf
git commit -m "Add new repo: testrepo"
git push origin master

# Go back to parent dir
cd ..

# Prepare and push new repo
mkdir testrepo
cd testrepo
git init
echo "Initial commit" > README.md
git add README.md
git commit -m "Initial commit"
git remote add origin "ssh://$GITOLITE_ADDR/testrepo.git"
git push -u origin master

# Create and push 3 branches
for i in {1..3}; do
  git checkout -b "branch$i"
  echo "Content for branch $i" > "file$i.txt"
  git add "file$i.txt"
  git commit -m "Commit on branch$i"
  git push origin "branch$i"
done

# Tag a release on master and push
git checkout master
git tag v1.0
git push origin v1.0

echo "Script completed successfully."
