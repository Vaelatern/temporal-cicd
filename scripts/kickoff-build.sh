#!/bin/sh

[ -z "$TCD_KICKOFF_URL" ] && echo "Need TCD_KICKOFF_URL set" && exit 1

printf "Enter repo alias: " && read nomen
printf "Branch to build: " && read branchname
printf "Build type [MakeBuildUpload]: " && read buildtype
printf "Any build type gotta be popped in the env var PATCH_OVERRIDE.\n"

[ -z "$branchname" ] && branchname=master
[ -z "$buildtype" ] && buildtype=MakeBuildUpload

curl -d "$(jq -n '$ARGS.named' \
			--arg repository "$nomen" \
			--arg ref "$branchname" \
			--arg build-pattern "$buildtype" \
			--arg compat-patch "$PATCH_OVERRIDE" \
			)" \
	-H "Authorization: Bearer a" \
	-X KICKOFF "${TCD_KICKOFF_URL}/${nomen}/${branchname}"
err1="$?"
echo
