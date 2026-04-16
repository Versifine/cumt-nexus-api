package config

import "time"

type Config struct {
	App    AppConfig
	DB     DBConfig
	HTTP   HTTPConfig
	Logger LoggerConfig
}
type AppConfig struct {
	Name string
	Env  string
}
type DBConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}
type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}
type LoggerConfig struct {
	Level  string
	Format string
}
type CacheConfig struct {
}
type AuthConfig struct {
}
