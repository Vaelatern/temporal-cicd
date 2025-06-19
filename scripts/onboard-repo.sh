#!/bin/sh

[ -z "$TCD_CACHE_URL" ] && echo "Need TCD_CACHE_URL set" && exit 1

printf "Enter repo URL: " && read URL
printf "SSH key contents should be provided via the env var SSH_KEY_CONTENTS.\n"
printf "Enter repo alias within our system: " && read nomen
printf "master branch name [master]: " && read branchname

[ -z "$branchname" ] && branchname=master

exec curl -d "$(jq -n '$ARGS.named' --arg url "$URL" --arg ssh-reading-private-key "$SSH_KEY_CONTENTS")" \
	-H "Authorization: Bearer a" \
	-X PUT "${TCD_CACHE_URL}/sync/${nomen}"
