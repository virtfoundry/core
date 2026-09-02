package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// DefaultConfigPath is the local-dev config file (gitignored). In-cluster pods use CONFIG_PATH.
const DefaultConfigPath = "config/config.yaml"

type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Logger        LoggerConfig        `mapstructure:"logger"`
	Security      SecurityConfig      `mapstructure:"security"`
	KubeVirt      KubeVirtConfig      `mapstructure:"kubevirt"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Networking    NetworkingConfig    `mapstructure:"networking"`
	Storage       StorageConfig       `mapstructure:"storage"`
}

type StorageConfig struct {
	DefaultClass      string `mapstructure:"default_class"`
	SnapshotClass     string `mapstructure:"snapshot_class"`
	WindowsBootSizeGi int    `mapstructure:"windows_boot_size_gi"`
	WindowsISOSizeGi  int    `mapstructure:"windows_iso_size_gi"`
}

type NetworkingConfig struct {
	Public   PublicNetworkConfig   `mapstructure:"public"`
	Isolated IsolatedNetworkConfig `mapstructure:"isolated"`
	VM       VMNetworkConfig       `mapstructure:"vm"`
}

type PublicNetworkConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	Mode         string   `mapstructure:"mode"`
	CIDR         string   `mapstructure:"cidr"`
	Gateway      string   `mapstructure:"gateway"`
	DNS          []string `mapstructure:"dns"`
	IPPoolStart  string   `mapstructure:"ip_pool_start"`
	IPPoolEnd    string   `mapstructure:"ip_pool_end"`
	BridgeName   string   `mapstructure:"bridge_name"`
	NADName      string   `mapstructure:"nad_name"`
	NADNamespace string   `mapstructure:"nad_namespace"`
}

type IsolatedNetworkConfig struct {
	BridgeName string `mapstructure:"bridge_name"`
}

type VMNetworkConfig struct {
	DefaultNetwork  string `mapstructure:"default_network"`
	AllowPodNetwork bool   `mapstructure:"allow_pod_network"`
}

type ObservabilityConfig struct {
	VelasExploreURL string `mapstructure:"velas_explore_url"`
}

type DatabaseConfig struct {
	// Driver selects the store backend: mysql (default when dsn set), memory, kubernetes.
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
	// Kubeconfig path for driver=kubernetes local dev; empty uses in-cluster or KUBECONFIG.
	Kubeconfig string `mapstructure:"kubeconfig"`
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
		Networking: NetworkingConfig{
			Isolated: IsolatedNetworkConfig{BridgeName: "virtfoundry-br0"},
			VM: VMNetworkConfig{
				DefaultNetwork:  "pod",
				AllowPodNetwork: true,
			},
		},
		Storage: StorageConfig{
			DefaultClass:      "local-path",
			WindowsBootSizeGi: 32,
			WindowsISOSizeGi:  8,
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
