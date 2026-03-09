#!/bin/bash
# Usage: ./add-token-permission.sh <token> <filepath>
# Example: ./add-token-permission.sh mytoken myrepo

set -euo pipefail

TOKEN="$1"
REPO="$2"
PERMISSION="KICKOFF /${REPO}/.*"
VAR_PATH="nomad/jobs/temporal-cicd/auth-keys/registration"

if [[ -z "$TOKEN" || -z "$REPO" ]]; then
    echo "Usage: $0 <token> <filepath>"
    exit 1
fi

echo "Fetching current variable..."
CURRENT=$(nomad var get -out json "$VAR_PATH")
MODIFY_INDEX=$(echo "$CURRENT" | jq -r '.ModifyIndex')
YAML_CONTENT=$(echo "$CURRENT" | jq -r '.Items."001" // ""')

if echo "$YAML_CONTENT" | grep -q "^${TOKEN}:"; then
    if echo "$YAML_CONTENT" | grep -qF "${PERMISSION}"; then
        echo "Permission already exists for token '$TOKEN': $PERMISSION"
        exit 0
    fi
    YAML_CONTENT=$(sed "s|^${TOKEN}:|${TOKEN}:\n - \"${PERMISSION}\"|" <<< "$YAML_CONTENT")
else
    if [[ -n "$YAML_CONTENT" ]]; then
        YAML_CONTENT="${YAML_CONTENT}
${TOKEN}:
 - \"${PERMISSION}\""
    else
        YAML_CONTENT="${TOKEN}:
 - \"${PERMISSION}\""
    fi
fi

echo "Adding permission '$PERMISSION' to token '$TOKEN'..."

nomad var put -check-index="$MODIFY_INDEX" "$VAR_PATH" \
    "001=$YAML_CONTENT"

echo "Done!"
