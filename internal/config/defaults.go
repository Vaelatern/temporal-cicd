package config

type Dir struct {
	Key           string `env:"TCD_DIR_KEY, default=../keys"`
	Cache         string `env:"TCD_DIR_CACHE, default=../cache"`
	CustomKickoff string `env:"TCD_DIR_CUSTOM_KICKOFF, default=../custom-kickoff"`
	RawArtifact   string `env:"TCD_DIR_ARTIFACT, default=../artifacts"`
	SSHKey        string `env:"TCD_DIR_SSHKEY, default=../ssh-keys"`
}

type Config struct {
	Dir      Dir
	CacheURL string `env:"TCD_CACHE_URL, default=http://localhost:8080"`
	Listen   string `env:"TCD_BIND, default=:8080"`
}
