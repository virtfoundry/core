package config

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	VM            VMConfig            `mapstructure:"vm"`
}

// VMConfig groups defaults applied at VM deploy time (used by template seeding).
type VMConfig struct {
	// DefaultPassword, when set, is baked into seed Linux templates that have no
	// userData of their own. The seed CloudInitNoCloud payload marks this as
	// expired (chpasswd.expire: True) so PAM forces the user to pick a new
	// password on first login. Empty means no password user-data is seeded —
	// Linux deploys must supply an SSH key or cloud_init_password (issue #97).
	// Override via config or VIRTFOUNDRY_VM_DEFAULT_PASSWORD.
	DefaultPassword string `mapstructure:"default_password"`
}

// BuiltinDefaultVMPassword is retained only for docs/tests that assert we no
// longer fall back to it. Do not use as a runtime default.
const BuiltinDefaultVMPassword = "ubuntu"

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
	// Driver selects the store backend: kubernetes (production) or memory (local dev/tests).
	Driver string `mapstructure:"driver"`
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
	// AllowedOrigins lists the browser origins allowed for CORS and for the
	// /ws/events and /ws/console WebSockets, e.g. "https://console.example.com".
	// The request host itself is always accepted because the UI is served
	// same-origin via the nginx proxy; this list is only needed when the UI
	// runs on a different origin than the API. Empty means fail closed for
	// cross-origin (no Access-Control-Allow-Origin: *).
	AllowedOrigins []string            `mapstructure:"allowed_origins"`
	LoginThrottle  LoginThrottleConfig `mapstructure:"login_throttle"`
	// ISOImport restricts where CDI may download tenant-supplied ISO URLs from.
	ISOImport ISOImportConfig `mapstructure:"iso_import"`
}

// ISOImportConfig is the admin allowlist for tenant-supplied ISO download URLs.
// HTTPS is always required and private, link-local and in-cluster targets are
// always refused, regardless of these settings. See docs/VM-TEMPLATES.md.
type ISOImportConfig struct {
	// AllowedHosts lists the hostnames CDI may fetch ISOs from; entries may be
	// wildcards ("*.blob.core.windows.net") matching subdomains. Empty keeps the
	// built-in defaults (importurl.DefaultAllowedHosts).
	AllowedHosts []string `mapstructure:"allowed_hosts"`
	// DisableHTTPImport refuses every URL-based ISO import. Tenants can still
	// register ISO templates from an existing volume (iso_volume_id).
	DisableHTTPImport bool `mapstructure:"disable_http_import"`
}

// LoginThrottleConfig tunes login brute-force protection. Zero values fall
// back to the auth package defaults (5 per username, 20 per IP, 10m window,
// 5m lockout).
type LoginThrottleConfig struct {
	UserMaxFailures int           `mapstructure:"user_max_failures"`
	IPMaxFailures   int           `mapstructure:"ip_max_failures"`
	Window          time.Duration `mapstructure:"window"`
	Lockout         time.Duration `mapstructure:"lockout"`
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
			JWTSecret: getEnv("JWT_SECRET", ""),
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
		VM: VMConfig{
			// Empty by default — never invent "ubuntu" (issue #97).
			DefaultPassword: getEnv("VIRTFOUNDRY_VM_DEFAULT_PASSWORD", ""),
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

const (
	// EnvISOAllowedHosts overrides security.iso_import.allowed_hosts with a
	// comma-separated list, for deployments whose config file is a read-only
	// ConfigMap.
	EnvISOAllowedHosts = "VIRTFOUNDRY_ISO_ALLOWED_HOSTS"
	// EnvISODisableHTTPImport set to "1" refuses every URL-based ISO import.
	EnvISODisableHTTPImport = "VIRTFOUNDRY_ISO_DISABLE_HTTP_IMPORT"
	// EnvAllowedOrigins overrides security.allowed_origins with a
	// comma-separated list of browser origins for CORS and WebSocket checks.
	EnvAllowedOrigins = "VIRTFOUNDRY_ALLOWED_ORIGINS"
)

// ApplyISOImportEnv lets operators set the ISO import allowlist via environment
// variables, which take precedence over the YAML config.
func ApplyISOImportEnv(cfg *Config) {
	if raw := os.Getenv(EnvISOAllowedHosts); raw != "" {
		var hosts []string
		for _, host := range strings.Split(raw, ",") {
			if host = strings.TrimSpace(host); host != "" {
				hosts = append(hosts, host)
			}
		}
		if len(hosts) > 0 {
			cfg.Security.ISOImport.AllowedHosts = hosts
		}
	}
	if os.Getenv(EnvISODisableHTTPImport) == "1" {
		cfg.Security.ISOImport.DisableHTTPImport = true
	}
}

// ApplyAllowedOriginsEnv lets operators set the CORS / WebSocket origin
// allowlist via environment, which takes precedence over the YAML config.
func ApplyAllowedOriginsEnv(cfg *Config) {
	raw := os.Getenv(EnvAllowedOrigins)
	if raw == "" {
		return
	}
	var origins []string
	for _, origin := range strings.Split(raw, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) > 0 {
		cfg.Security.AllowedOrigins = origins
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
