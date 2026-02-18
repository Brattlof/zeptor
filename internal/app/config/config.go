package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Routing   RoutingConfig   `mapstructure:"routing"`
	EBPF      EBPFConfig      `mapstructure:"ebpf"`
	Rendering RenderingConfig `mapstructure:"rendering"`
	Build     BuildConfig     `mapstructure:"build"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

type AppConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type RoutingConfig struct {
	AppDir    string `mapstructure:"appDir"`
	PublicDir string `mapstructure:"publicDir"`
}

type EBPFConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	CacheSize int    `mapstructure:"cacheSize"`
	CacheTTLS int    `mapstructure:"cacheTTLSec"`
	Interface string `mapstructure:"interface"`
}

type RenderingConfig struct {
	Mode           string `mapstructure:"mode"`
	ISRRevalidateS int    `mapstructure:"isrRevalidateSec"`
}

type BuildConfig struct {
	OutDir    string `mapstructure:"outDir"`
	StaticDir string `mapstructure:"staticDir"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("zeptor")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/zeptor")
	}

	v.AutomaticEnv()
	v.SetEnvPrefix("ZEPTOR")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	if envPath := os.Getenv("ZEPTOR_CONFIG"); envPath != "" {
		v.SetConfigFile(envPath)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("error reading env config: %w", err)
		}
	}

	if ebpfEnv := os.Getenv("ZEPTOR_EBPF_ENABLED"); ebpfEnv != "" {
		v.Set("ebpf.enabled", ebpfEnv == "true")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.port", 3000)
	v.SetDefault("app.host", "0.0.0.0")

	v.SetDefault("routing.appDir", "./app")
	v.SetDefault("routing.publicDir", "./public")

	v.SetDefault("ebpf.enabled", true)
	v.SetDefault("ebpf.interface", "eth0")
	v.SetDefault("ebpf.cacheSize", 10000)
	v.SetDefault("ebpf.cacheTTLSec", 60)

	v.SetDefault("rendering.mode", "ssr")
	v.SetDefault("rendering.isrRevalidateSec", 300)

	v.SetDefault("build.outDir", "./.zeptor")
	v.SetDefault("build.staticDir", "./dist")

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}

func (c *Config) CacheTTL() time.Duration {
	return time.Duration(c.EBPF.CacheTTLS) * time.Second
}

func (c *Config) ISRRevalidate() time.Duration {
	return time.Duration(c.Rendering.ISRRevalidateS) * time.Second
}
