# SSH Key Management for Gitolite Integration

This directory contains enhanced solutions for managing SSH keys when integrating gitolite with temporalio-cicd. 

## Problem Overview

The basic hooks generate SSH keys but don't properly register them with gitolite's authentication system. Gitolite needs to know about these keys to allow the Cache service to access repositories securely.

## Solution Approaches

### Approach 1: Gitolite-Admin Integration (Recommended)

**File**: `repo-creation-hook-enhanced`

This approach uses gitolite's standard mechanism by modifying the `gitolite-admin` repository:

**Features**:
- Creates dedicated CI user for each repository (`ci-$REPO_NAME`)
- Adds SSH public key to `keydir/ci-$REPO_NAME@ci.pub`
- Updates `gitolite.conf` with repository-specific access
- Commits and pushes changes to gitolite-admin
- Automatically rolls back on failure

**Workflow**:
1. Generate SSH key pair
2. Clone gitolite-admin repository
3. Add public key to `keydir/`
4. Update `gitolite.conf` with user permissions
5. Commit and push changes
6. Register with Cache service

**Pros**:
- Uses standard gitolite mechanisms
- Clean separation of concerns
- Easy to audit via git history
- Follows gitolite best practices

**Cons**:
- Requires write access to gitolite-admin
- Slightly more complex setup

### Approach 2: Per-Repository Configuration

**Files**: `repo-creation-hook-config` and `ssh-auth-keys`

This approach stores SSH keys in each repository's git configuration and uses a custom authentication script:

**Features**:
- Stores SSH keys in repository git config
- Custom `ssh-auth-keys` script for key management
- Updates global `authorized_keys` file
- Supports key listing, registration, and revocation

**Workflow**:
1. Generate SSH key pair
2. Store public key in repository git config
3. Register with custom auth system
4. Update global authorized_keys file
5. Register with Cache service

**Pros**:
- No gitolite-admin modifications required
- Centralized key management
- Easy key rotation and revocation
- Works with existing gitolite installations

**Cons**:
- Requires custom script deployment
- More complex infrastructure
- Additional maintenance overhead

## Installation Instructions

### Approach 1: Gitolite-Admin Integration

1. **Make the hook executable**:
```bash
chmod +x repo-creation-hook-enhanced
```

2. **Configure environment variables**:
```bash
export GITOLITE_ADMIN_REPO="$HOME/repositories/gitolite-admin.git"
export GITOLITE_ADMIN_WORKDIR="/tmp/gitolite-admin"
export LRCICD_CACHE_URL="http://your-cache-service:8081"
export LRCICD_REGISTER_TOKEN="your-registration-token"
```

3. **Update gitolite.conf**:
```conf
repo @all
    config hooks.repo-creation = /path/to/repo-creation-hook-enhanced %GL_REPO
```

### Approach 2: Per-Repository Configuration

1. **Install the custom auth script**:
```bash
chmod +x ssh-auth-keys
sudo cp ssh-auth-keys /usr/local/libexec/gitolite/ssh-auth-keys
sudo mkdir -p /usr/local/libexec/gitolite
```

2. **Make the hook executable**:
```bash
chmod +x repo-creation-hook-config
```

3. **Configure environment variables**:
```bash
export SSH_AUTH_COMMAND="/usr/local/libexec/gitolite/ssh-auth-keys"
export GITOLITE_REPO_BASE="$HOME/repositories"
export LRCICD_CACHE_URL="http://your-cache-service:8081"
export LRCID_REGISTER_TOKEN="your-registration-token"
```

4. **Update gitolite.conf**:
```conf
repo @all
    config hooks.repo-creation = /path/to/repo-creation-hook-config %GL_REPO
```

## Key Management Commands (Approach 2)

The `ssh-auth-keys` script provides comprehensive key management:

```bash
# Register a new key
ssh-auth-keys register my-repo key-001 "ssh-rsa AAAAB3... user@host"

# List all keys for a repository
ssh-auth-keys list my-repo

# Check if a key is authorized
ssh-auth-keys auth my-repo key-001

# Remove a key
ssh-auth-keys unregister my-repo key-001

# Update global authorized_keys file
ssh-auth-keys update-authorized-keys
```

## Security Considerations

### Approach 1 Security:
- Follows gitolite's built-in security model
- Keys are managed through gitolite's access control
- Audit trail via git history
- Each repository gets isolated CI user

### Approach 2 Security:
- Keys stored in git config (still accessible but distributed)
- Custom script needs proper permissions
- Global authorized_keys file management
- Additional attack surface with custom scripts

## Recommendation

**Use Approach 1 (Gitolite-Admin Integration)** for most cases because:
- It uses gitolite's standard security model
- No custom infrastructure required
- Easier to audit and maintain
- Better isolation between repositories

**Use Approach 2 (Per-Repository Configuration)** only if:
- You cannot modify gitolite-admin
- You need centralized key management across many repositories
- You have specific requirements for key rotation/management

## Migration

If starting with Approach 1 and later needing Approach 2:
1. Existing keys remain in gitolite-admin
2. New repositories use the custom system
3. Gradually migrate by updating repository configs

Both approaches are compatible with the existing post-receive hooks and Cache service integration.