package config

import "time"

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	HTTP     HTTPConfig
	Log      LogConfig
	Auth     AuthConfig
	Storage  ObjectStorageConfig
	Upload   UploadConfig
}
type AppConfig struct {
	Name           string
	Env            string
	StartupTimeout time.Duration
}
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	MaxConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}
type HTTPConfig struct {
	Addr               string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ShutdownTimeout    time.Duration
	CORSAllowedOrigins []string
}
type LogConfig struct {
	Level  string
	Format string
}
type CacheConfig struct {
}
type AuthConfig struct {
	TokenSecret    string
	AccessTokenTTL time.Duration
}
type ObjectStorageConfig struct {
	Provider        string
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	PublicBaseURL   string
	ForcePathStyle  bool
	LocalRoot       string
}
type UploadConfig struct {
	ImageMaxBytes           int
	ImageMaxCountPerPost    int
	ImageMaxCountPerComment int
}
