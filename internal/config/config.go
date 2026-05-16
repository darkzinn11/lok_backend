package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv          string
	Host            string
	Port            string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	DatabaseURL     string
	JWTSecret          string
	CORSAllowedOrigins []string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL time.Duration
	AWSEndpoint     string
	AWSAccessKey    string
	AWSSecretKey    string
	AWSRegion       string
	AWSBucket       string
	AWSUseSSL       bool
	UploadDir       string
	LogDir          string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, loading from environment variables")
	}

	cfg := &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		Host:            getEnv("HOST", "127.0.0.1"),
		Port:            getEnv("PORT", "8080"),
		DBHost:          getEnv("DB_HOST", "postgres"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "lokdb"),
		DBPassword:      getEnv("DB_PASSWORD", "lokpassword"),
		DBName:          getEnv("DB_NAME", "lokcenter"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		JWTSecret:          strings.TrimSpace(os.Getenv("JWT_SECRET")),
		CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		AccessTokenTTL:     getEnvAsDuration("JWT_ACCESS_TTL", 2*time.Hour),
		RefreshTokenTTL: getEnvAsDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		AWSEndpoint:     getEnv("AWS_ENDPOINT", "http://localhost:9000"),
		AWSAccessKey:    getEnv("AWS_ACCESS_KEY_ID", "admin_s3"),
		AWSSecretKey:    getEnv("AWS_SECRET_ACCESS_KEY", "admin_secret_s3"),
		AWSRegion:       getEnv("AWS_REGION", "us-east-1"),
		AWSBucket:       getEnv("AWS_BUCKET", "lokcenter-bucket"),
		AWSUseSSL:       getEnvAsBool("AWS_USE_SSL", false),
		UploadDir:       getEnv("UPLOAD_DIR", "/srv/apps/lokcenter/uploads"),
		LogDir:          getEnv("LOG_DIR", "/srv/apps/lokcenter/logs"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	var errs []error

	if c.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required"))
	}

	if c.AccessTokenTTL <= 0 {
		errs = append(errs, errors.New("JWT_ACCESS_TTL must be greater than zero"))
	}

	if c.RefreshTokenTTL <= 0 {
		errs = append(errs, errors.New("JWT_REFRESH_TTL must be greater than zero"))
	}

	if c.Port == "" {
		errs = append(errs, errors.New("PORT is required"))
	}

	if c.DBHost == "" || c.DBPort == "" || c.DBUser == "" || c.DBName == "" {
		errs = append(errs, errors.New("database connection settings are incomplete"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	str := getEnv(key, "")
	if value, err := strconv.Atoi(str); err == nil {
		return value
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	str := getEnv(key, "")
	if str == "" {
		return fallback
	}

	value, err := time.ParseDuration(str)
	if err != nil {
		return fallback
	}

	return value
}

func getEnvAsBool(key string, fallback bool) bool {
	str := getEnv(key, "")
	if value, err := strconv.ParseBool(str); err == nil {
		return value
	}
	return fallback
}

func getEnvAsSlice(key string, fallback []string) []string {
	str := getEnv(key, "")
	if str == "" {
		return fallback
	}
	parts := strings.Split(str, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
