package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string     `yaml:"env" env-default:"prod"`
	PostgreSQL PostgreSQL `yaml:"postgresql"`
	HTTPServer HTTPServer `yaml:"http_server"`
	JWT        JWT        `yaml:"jwt"`
	SMTP       SMTP       `yaml:"smtp"`
	Minio      Minio      `yaml:"minio"`
}

type PostgreSQL struct {
	Host     string `yaml:"host" env-required:"true"`
	Port     string `yaml:"port" env-required:"true"`
	Username string `yaml:"username" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	Database string `yaml:"database" env-required:"true"`
}

type HTTPServer struct {
	Address          string        `yaml:"address" env-required:"true"`
	Timeout          time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout      time.Duration `yaml:"idle_timeout" env-default:"60s"`
	AllowedOrigins   []string      `yaml:"allowed_origins" env-default:"*"`
	AllowCredentials bool          `yaml:"allow_credentials"`
	AllowedMethods   []string      `yaml:"allowed_methods" env-default:"*"`
	AllowedHeaders   []string      `yaml:"allowed_headers" env-default:"*"`
	StaticURL        string        `yaml:"static_url" env-required:"true"`
}

type JWT struct {
	Secret          string        `yaml:"secret" env-required:"true"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl" env-required:"true"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env-required:"true"`
}

type SMTP struct {
	Host     string `yaml:"host" env-required:"true"`
	Port     string `yaml:"port" env-required:"true"`
	Username string `yaml:"username" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
}

type Minio struct {
	Endpoint        string `yaml:"endpoint" env-required:"true"`
	AccessKeyID     string `yaml:"access_key_id" env-required:"true"`
	SecretAccessKey string `yaml:"secret_access_key" env-required:"true"`
	UseSSL          bool   `yaml:"use_ssl"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadByPath(configPath)
}

func MustLoadByPath(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("config reading error: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
