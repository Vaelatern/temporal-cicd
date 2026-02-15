# Gitolite Hooks for temporalio-cicd Integration

This directory contains example gitolite hooks that integrate a gitolite server with the temporalio-cicd service for automated CI/CD workflows.

## Hook Files

### 1. `repo-creation-hook` (Repository Creation Hook)

**Purpose**: Runs when a new repository is created in gitolite. It registers the repository with the Cache service and generates authentication tokens.

**Features**:
- Registers repository with Cache service via `PUT /sync/$reponame`
- Uses `$LRCICD_REGISTER_TOKEN` for authentication
- Generates SSH key pair for secure repository access
- Generates CI token for subsequent CI/CD operations
- Stores CI token in git repository metadata
- Uses configurable cache base URL via `$LRCICD_CACHE_URL`

**Usage in Gitolite**:
```bash
# Add to gitolite.conf in the repo creation section:
repo @all
    config hooks.repo-creation = /path/to/repo-creation-hook %GL_REPO
```

**Environment Variables**:
- `LRCICD_CACHE_URL`: Base URL of the Cache service (default: `http://gitolite-cache:8081`)
- `LRCICD_REGISTER_TOKEN`: Authentication token for Cache service registration

### 2. `post-receive-hook` (Post-Receive Hook)

**Purpose**: Runs after code is pushed to a repository. It triggers CI/CD workflows using the `.vaelci.json` configuration.

**Features**:
- Triggers CI/CD via `KICKOFF /$repository/$ref` 
- Uses the CI token stored during repository creation
- Reads `.vaelci.json` from the pushed commit
- Supports branch-specific workflows
- Only processes branches (skips tags and branch deletions)
- Uses configurable kickoff base URL via `$LRCICD_KICKOFF_URL`

**Usage in Gitolite**:
```bash
# Add to gitolite.conf for repositories that need CI/CD:
repo @ci-enabled
    config hooks.post-receive = /path/to/post-receive-hook
```

**Environment Variables**:
- `LRCICD_KICKOFF_URL`: Base URL of the Kickoff service (default: `http://gitolite-kickoff:8083`)

## Installation Instructions

### 1. Make Hooks Executable
```bash
chmod +x repo-creation-hook
chmod +x post-receive-hook
```

### 2. Configure Gitolite

Add to your `gitolite.conf`:

```conf
# Enable repo creation hook for all repositories
repo @all
    config hooks.repo-creation = /path/to/repo-creation-hook %GL_REPO

# Enable CI/CD for specific repositories
repo @ci-enabled
    config hooks.post-receive = /path/to/post-receive-hook

# Or enable for all repos
repo @all
    config hooks.post-receive = /path/to/post-receive-hook
```

### 3. Set Environment Variables

In your gitolite user's environment or gitolite.rc:

```bash
export LRCICD_CACHE_URL="http://your-cache-service:8081"
export LRCICD_KICKOFF_URL="http://your-kickoff-service:8083"
export LRCICD_REGISTER_TOKEN="your-secure-registration-token"
```

### 4. Configure Service Authentication

Ensure your Cache service has the `$LRCICD_REGISTER_TOKEN` configured in its `keys.d/` directory with permission to access `PUT /sync/*`.

## Workflow

1. **Repository Creation**: When a new repository is created in gitolite, the `repo-creation-hook`:
   - Generates SSH keys for repository access
   - Registers the repository with the Cache service
   - Stores a CI token in git metadata

2. **Code Push**: When code is pushed to any enabled repository:
   - The `post-receive-hook` reads the pushed `.vaelci.json`
   - Triggers the CI/CD workflow using the stored CI token
   - The Kickoff service starts the temporal workflow

## Example .vaelci.json

```json
{
    "build-pattern": "MakeBuildUpload"
}
```

## Security Notes

- The registration token (`$LRCICD_REGISTER_TOKEN`) should be kept secure
- CI tokens are stored in git repository metadata and are specific to each repository
- SSH keys are generated per repository and stored content-addressed in the Cache service
- All hooks validate responses before proceeding