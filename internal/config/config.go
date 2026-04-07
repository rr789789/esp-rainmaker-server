package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Admin    AdminConfig    `mapstructure:"admin"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	OTA      OTAConfig      `mapstructure:"ota"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	AccessTokenTTL  int    `mapstructure:"access_token_ttl"`
	RefreshTokenTTL int    `mapstructure:"refresh_token_ttl"`
	IDTokenTTL      int    `mapstructure:"id_token_ttl"`
}

type AdminConfig struct {
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	SessionTTL int    `mapstructure:"session_ttl"`
}

type LoggingConfig struct {
	Level           string `mapstructure:"level"`
	APILogEnabled   bool   `mapstructure:"api_log_enabled"`
	APILogMax int    `mapstructure:"api_log_max_entries"`
}

type OTAConfig struct {
	FirmwareDir string `mapstructure:"firmware_dir"`
}

var AppConfig *Config

func Load(configPath string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/rainmaker")
	}

	viper.SetEnvPrefix("RAINMAKER")
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is OK, use defaults
		fmt.Printf("Note: config file not found, using defaults\n")
	}

	AppConfig = &Config{}
	return viper.Unmarshal(AppConfig)
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.path", "./data/rainmaker.db")
	viper.SetDefault("jwt.secret", "change-this-secret-in-production")
	viper.SetDefault("jwt.access_token_ttl", 3600)
	viper.SetDefault("jwt.refresh_token_ttl", 2592000)
	viper.SetDefault("jwt.id_token_ttl", 3600)
	viper.SetDefault("admin.username", "admin")
	viper.SetDefault("admin.password", "admin123")
	viper.SetDefault("admin.session_ttl", 86400)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.api_log_enabled", true)
	viper.SetDefault("logging.api_log_max_entries", 1000)
	viper.SetDefault("ota.firmware_dir", "./data/firmware")
}
