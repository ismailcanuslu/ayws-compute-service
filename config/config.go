package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Proxmox    ProxmoxConfig    `mapstructure:"proxmox"`
	Docker     DockerConfig     `mapstructure:"docker"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Serverless ServerlessConfig `mapstructure:"serverless"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	BodyLimit    int           `mapstructure:"body_limit"`
}

type ProxmoxConfig struct {
	Host        string `mapstructure:"host"`
	TokenID     string `mapstructure:"token_id"`
	TokenSecret string `mapstructure:"token_secret"`
	InsecureTLS bool   `mapstructure:"insecure_tls"`
	Node        string `mapstructure:"node"`
}

type DockerConfig struct {
	Host       string `mapstructure:"host"`
	APIVersion string `mapstructure:"api_version"`
}

type KubernetesConfig struct {
	Kubeconfig string `mapstructure:"kubeconfig"`
	Namespace  string `mapstructure:"namespace"`
}

type ServerlessConfig struct {
	RuntimeImage    string `mapstructure:"runtime_image"`
	TimeoutSeconds  int    `mapstructure:"timeout_seconds"`
	MemoryLimitMB   int    `mapstructure:"memory_limit_mb"`
}

// Load reads config.yaml (or env overrides) and returns a Config struct.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// ENV override: COMPUTE_SERVER_PORT → server.port
	v.SetEnvPrefix("COMPUTE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("server.port", 8001)
	v.SetDefault("server.body_limit", 32)
	v.SetDefault("docker.host", "unix:///var/run/docker.sock")
	v.SetDefault("docker.api_version", "1.43")
	v.SetDefault("kubernetes.namespace", "default")
	v.SetDefault("serverless.timeout_seconds", 30)
	v.SetDefault("serverless.memory_limit_mb", 128)
	v.SetDefault("serverless.runtime_image", "python:3.11-slim")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
