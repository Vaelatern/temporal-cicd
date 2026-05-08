package config

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Dir    Dir         `yaml:"dir,omitempty" json:"dir,omitempty" toml:"dir,omitempty"`
	Cache  CacheConfig `yaml:"cache,omitempty" json:"cache,omitempty" toml:"cache,omitempty"`
	Listen string      `yaml:"listen,omitempty" json:"listen,omitempty" toml:"listen,omitempty"`
}

type Dir struct {
	Key string `env:"TCD_DIR_KEY, overwrite" yaml:"key,omitempty"
					json:"key,omitempty"
					toml:"key,omitempty"`
	Cache string `env:"TCD_DIR_CACHE, overwrite"
					yaml:"cache,omitempty"
					json:"cache,omitempty"
					toml:"cache,omitempty"`
	CustomKickoff string `env:"TCD_DIR_CUSTOM_KICKOFF, overwrite"
					yaml:"custom_kickoff,omitempty"
					json:"custom_kickoff,omitempty"
					toml:"custom_kickoff,omitempty"`
	RawArtifact string `env:"TCD_DIR_ARTIFACT, overwrite"
					yaml:"raw_artifact,omitempty"
					json:"raw_artifact,omitempty"
					toml:"raw_artifact,omitempty"`
	SSHKey string `env:"TCD_DIR_SSHKEY, overwrite"
					yaml:"ssh_key,omitempty"
					json:"ssh_key,omitempty"
					toml:"ssh_key,omitempty"`
}

type CacheConfig struct {
	URL string `env:"TCD_CACHE_URL, overwrite"
					yaml:"url,omitempty"
					json:"url,omitempty"
					toml:"url,omitempty"`
	Headers map[string]string `env:"TCD_CACHE_HEADERS, overwrite"
					yaml:"headers,omitempty"
					json:"headers,omitempty"
					toml:"headers,omitempty"`
}

func LoadConfig() (*Config, error) {
	var conf Config

	// Set defaults
	conf.Dir.Key = "../keys"
	conf.Dir.Cache = "../cache"
	conf.Dir.CustomKickoff = "../custom-kickoff"
	conf.Dir.RawArtifact = "../artifacts"
	conf.Dir.SSHKey = "../ssh-keys"
	conf.Cache.URL = "http://localhost:8080"
	conf.Listen = ":8080"

	// Load config file(s)
	configFile := os.Getenv("TCD_CONFIG_FILE")
	if configFile == "" {
		configFile = "../config.yaml"
	}

	info, err := os.Stat(configFile)
	// don't check err except to skip loading if stat didn't find anything
	if err == nil && info.IsDir() {
		// Load directory of config files
		cfg, err := loadConfigDir(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config directory %s: %w", configFile, err)
		}
		if cfg != nil {
			mergeConfig(&conf, *cfg)
		}
	} else if err == nil {
		// Load single config file
		cfg, err := loadSingleConfigFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
		if cfg != nil {
			mergeConfig(&conf, *cfg)
		}
	} else { // err != nil
		log.Printf("Config file %s was not loaded - not found\n", configFile)
	}

	// Apply environment variables last (highest priority)
	if err := envconfig.Process(context.Background(), &conf); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	return &conf, nil
}

func loadConfigDir(dirPath string) (*Config, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var merged Config

	// os.ReadDir returns entries sorted by filename (lexicographic order)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" && ext != ".toml" {
			continue
		}

		filePath := filepath.Join(dirPath, name)
		cfg, err := loadSingleConfigFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", filePath, err)
		}

		if cfg != nil {
			mergeConfig(&merged, *cfg)
		}
	}

	return &merged, nil
}

func loadSingleConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml", ".json":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML/JSON config file: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse TOML config file: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json, .toml)", ext)
	}

	return &cfg, nil
}

// mergeConfig is overly hardcoded but does merge the configurations down
func mergeConfig(dst *Config, src Config) {
	if src.Dir.Key != "" {
		dst.Dir.Key = src.Dir.Key
	}
	if src.Dir.Cache != "" {
		dst.Dir.Cache = src.Dir.Cache
	}
	if src.Dir.CustomKickoff != "" {
		dst.Dir.CustomKickoff = src.Dir.CustomKickoff
	}
	if src.Dir.RawArtifact != "" {
		dst.Dir.RawArtifact = src.Dir.RawArtifact
	}
	if src.Dir.SSHKey != "" {
		dst.Dir.SSHKey = src.Dir.SSHKey
	}

	if src.Cache.URL != "" {
		dst.Cache.URL = src.Cache.URL
	}

	if src.Cache.Headers != nil {
		if dst.Cache.Headers == nil {
			dst.Cache.Headers = make(map[string]string)
		}
		for k, v := range src.Cache.Headers {
			dst.Cache.Headers[k] = v
		}
	}

	if src.Listen != "" {
		dst.Listen = src.Listen
	}
}

func (c *Config) AddCacheHeaders(req *http.Request) {
	for key, value := range c.Cache.Headers {
		req.Header.Set(key, value)
	}
}
