package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server         ServerConfig         `mapstructure:"server"`
	Logger         LoggerConfig         `mapstructure:"logger"`
	Security       SecurityConfig       `mapstructure:"security"`
	KubeVirt       KubeVirtConfig       `mapstructure:"kubevirt"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Observability  ObservabilityConfig  `mapstructure:"observability"`
}

type ObservabilityConfig struct {
	VelasExploreURL string `mapstructure:"velas_explore_url"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type KubeVirtConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Kubeconfig string `mapstructure:"kubeconfig"`
	Namespace  string `mapstructure:"namespace"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type SecurityConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
	JWTExpire int    `mapstructure:"jwt_expire"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Logger: LoggerConfig{
			Level:  "info",
			Format: "json",
		},
		Security: SecurityConfig{
			JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
			JWTExpire: 86400,
		},
		KubeVirt: KubeVirtConfig{
			Enabled:    true,
			Kubeconfig: getEnv("KUBECONFIG", ""),
			Namespace:  getEnv("KUBEVIRT_NAMESPACE", "default"),
		},
	}
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
