package config

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

type DebugLevel string

const DebugLevelNone DebugLevel = ""
const DebugLevelDebug DebugLevel = "debug"
const DebugLevelShowSensitive DebugLevel = "show-sensitive"

func parseDebugLevel(s string) DebugLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "false", "0", "no", "off", "none", "nil":
		return DebugLevelNone
	case "show-sensitive":
		return DebugLevelShowSensitive
	default: // true, debug, True, and most other values
		return DebugLevelDebug
	}
}

func (d *DebugLevel) UnmarshalText(text []byte) error {
	*d = parseDebugLevel(string(text))
	return nil
}

func (d *DebugLevel) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*d = DebugLevelNone
		return nil
	}
	var v any
	if err := value.Decode(&v); err != nil {
		return err
	}
	switch x := v.(type) {
	case nil:
		*d = DebugLevelNone
	case bool:
		if x {
			*d = DebugLevelDebug
		} else {
			*d = DebugLevelNone
		}
	case string:
		*d = parseDebugLevel(x)
	case int:
		if x == 0 {
			*d = DebugLevelNone
		} else {
			*d = DebugLevelDebug
		}
	default:
		*d = DebugLevelDebug
	}
	return nil
}

type Config struct {
	Debug         DebugLevel          `env:"TCD_DEBUG, overwrite" yaml:"debug,omitempty" json:"debug,omitempty" toml:"debug,omitempty"`
	Dir           Dir                 `yaml:"dir,omitempty" json:"dir,omitempty" toml:"dir,omitempty"`
	Cache         CacheConfig         `yaml:"cache,omitempty" json:"cache,omitempty" toml:"cache,omitempty"`
	Listen        string              `yaml:"listen,omitempty" json:"listen,omitempty" toml:"listen,omitempty"`
	SharedSecrets SharedSecretsConfig `yaml:"shared_secrets,omitempty" json:"shared_secrets,omitempty" toml:"shared_secrets,omitempty"`
}

type Dir struct {
	SharedSecrets string `env:"TCD_DIR_SHAREDSECRETS, overwrite" yaml:"shared-secrets,omitempty"
					json:"shared-secrets,omitempty"
					toml:"shared-secrets,omitempty"`
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

type SharedSecretsConfig struct {
	Dir string `env:"TCD_DIR_SHAREDSECRETS, overwrite" yaml:"dir,omitempty" json:"dir,omitempty" toml:"dir,omitempty"`
}

func LoadConfig() (*Config, error) {
	var conf Config

	// Set defaults
	conf.Dir.SharedSecrets = "../shared-secrets"
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

	if conf.Debug != DebugLevelNone {
		conf.dumpWorkingConfig()
	}

	return &conf, nil
}

func (c *Config) dumpWorkingConfig() {
	out := *c
	if c.Debug != DebugLevelShowSensitive && len(c.Cache.Headers) > 0 {
		out.Cache.Headers = make(map[string]string, len(c.Cache.Headers))
		for k := range c.Cache.Headers {
			out.Cache.Headers[k] = "***"
		}
	}
	fmt.Printf("boot_time: %s\nworking config:\n", time.Now().Format(time.RFC3339Nano))
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	_ = enc.Encode(out)
	_ = enc.Close()
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
	if src.Debug != "" {
		dst.Debug = src.Debug
	}
	if src.Dir.SharedSecrets != "" {
		dst.Dir.SharedSecrets = src.Dir.SharedSecrets
	}
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

// DebugMiddleware logs every request when debug is on.
// Tokens appear only at show-sensitive, via basicauth.GetToken.
func (c *Config) DebugMiddleware(next http.Handler) http.Handler {
	if c.Debug == DebugLevelNone {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Debug == DebugLevelShowSensitive {
			log.Printf("[debug] %s %s from %s token=%q", r.Method, r.URL.Path, r.RemoteAddr, basicauth.GetToken(r))
		} else {
			log.Printf("[debug] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
		next.ServeHTTP(w, r)
	})
}
