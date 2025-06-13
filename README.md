# CI CD leveraging Temporal Workflows

This is several components that work together.

## The Components

All http endpoints have a token issued and written to a configuration file directory found at `/keys.d/` inside the containers with the http servers. The `keys.d` file format is a yaml dicionary, where the key is a token to be presented to conduct an action, followed by a list of regular expressions that match what the token is allowed to do (including HTTP method and path). This directory is rescanned on every request, with an in-memory cache of 60 seconds.

### Cache

Our ability to clone a repo may depend on the upstream existing. Let's have a cache so we can get updates and keep references (tags, branches), locally.

To do that we have a small go program that sits on top of a disk. That disk stores all the various refs for all the cloned repositories that we are told about. These are stored in a single bare git repository using a git namespace per source repository to deduplicate storage as much as reasonable.

To tell the go program to update and make sure the ref is up to date, hit the endpoint: POST `/sync/$reponame/$ref`.

To add a repo to the cache, hit `PUT /sync/$reponame` with the body being a json object as such: `{"url": "git@github.com:owner/repo.git", "ssh-reading-private-key": "SSH_KEY_HERE"}`. SSH keys are stored content-addressed (the sha256sum of the ssh key is the filename, and the repository is configured to use that ssh key for any updates.

To fetch from the cache, use GET `/download/$reponame/$ref.tar.gz` and git archive will produce the latest tarball of the reference for download.

The Cache is two parts. One is a go program, and the other is a dockerfile running the one program with git installed, and a volume mount at `/repos` for the git repositories we cache.

### Artifacts

This is a tiny go program on top of a volume. The abstraction used in all our go programs for filesystem access is fs.FS, to allow portability. To install an artifact, PUT `/$path` and the $path can have multiple slashes in it. To get the artifact back, it's simply GET `/$path`. On GET, the "modification time" of the file is updated to allow a cleaning process to remove obsolete artifacts.

All content is stored hash-addressed, with the sha256sum pointing at the entry, and the relevant paths storing only the reference to the filename

You can also upload containers to Artifacts and use it as an OCI registry.

All PUT calls have the variable `reftype` with them. If the reftype is `branch` then the reftype must always be `branch`. If the `reftype` is `tag` then the uploaded artifact can not be changed later, and is marked immutable and all future PUT calls will fail.

### Builder

This is a temporal enabled go application that pulls a tarball from the Cache service, runs `make build` in the directory (along with any environment variables that were passed in in the Env key), runs `make upload` to upload, and then runs `make deploy`. Finally the working directory is fully removed. The clone, build, upload, deploy, and clean steps are performed with strict affinity so the same worker gets all of them.

### Kickoff

This is a hook that is installed on a git repository that calls KICKOFF /$repository/$ref when a push happens to the git repository. Like all other HTTP actions, this requires a bearer token installed. This triggers the overall building temporal workflow. When a subsequent KICKOFF is received for a workflow already in progress, the workflow is sent a signal. The workflow then cancels any existing build, and re-runs from the cache command.
