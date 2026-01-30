package db

import (
	"errors"
	"fmt"
	"os"
)

// ErrMissingPassword is returned when required password environment variables are not set.
var ErrMissingPassword = errors.New("required password environment variable not set")

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	MongoHost     string
	MongoPort     string
	MongoUser     string
	MongoPassword string
	MongoDB       string

	NATSHost string
	NATSPort string

	RedisHost     string
	RedisPort     string
	RedisPassword string
}

// isDevelopmentMode returns true if ASGARD_ENV is set to "development".
func isDevelopmentMode() bool {
	return os.Getenv("ASGARD_ENV") == "development"
}

// LoadConfig loads database configuration from environment variables.
// In production mode, password environment variables are required and will
// cause an error if not set. In development mode, default values are used.
func LoadConfig() (*Config, error) {
	isDev := isDevelopmentMode()

	// Get passwords - required in production, defaults in development
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	mongoPassword := os.Getenv("MONGO_PASSWORD")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// In production, require passwords to be explicitly set
	if !isDev {
		var missing []string
		if postgresPassword == "" {
			missing = append(missing, "POSTGRES_PASSWORD")
		}
		if mongoPassword == "" {
			missing = append(missing, "MONGO_PASSWORD")
		}
		if redisPassword == "" {
			missing = append(missing, "REDIS_PASSWORD")
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("%w: %v (set ASGARD_ENV=development to use defaults)", ErrMissingPassword, missing)
		}
	} else {
		// Development mode: use defaults if not set (but log a warning)
		if postgresPassword == "" {
			postgresPassword = "dev_postgres_password"
			fmt.Println("[CONFIG] WARNING: Using default POSTGRES_PASSWORD for development")
		}
		if mongoPassword == "" {
			mongoPassword = "dev_mongo_password"
			fmt.Println("[CONFIG] WARNING: Using default MONGO_PASSWORD for development")
		}
		if redisPassword == "" {
			redisPassword = "dev_redis_password"
			fmt.Println("[CONFIG] WARNING: Using default REDIS_PASSWORD for development")
		}
	}

	cfg := &Config{
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "55432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: postgresPassword,
		PostgresDB:       getEnv("POSTGRES_DB", "asgard"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		MongoHost:     getEnv("MONGO_HOST", "localhost"),
		MongoPort:     getEnv("MONGO_PORT", "27018"),
		MongoUser:     getEnv("MONGO_USER", "admin"),
		MongoPassword: mongoPassword,
		MongoDB:       getEnv("MONGO_DB", "asgard"),

		NATSHost: getEnv("NATS_HOST", "localhost"),
		NATSPort: getEnv("NATS_PORT", "4222"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: redisPassword,
	}

	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresDB,
		c.PostgresSSLMode,
	)
}

func (c *Config) MongoURI() string {
	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/%s?authSource=admin",
		c.MongoUser,
		c.MongoPassword,
		c.MongoHost,
		c.MongoPort,
		c.MongoDB,
	)
}

func (c *Config) NATSURI() string {
	return fmt.Sprintf("nats://%s:%s", c.NATSHost, c.NATSPort)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

// RedisURL returns a Redis connection URL with authentication if configured.
func (c *Config) RedisURL() string {
	if c.RedisPassword != "" {
		return fmt.Sprintf("redis://:%s@%s:%s", c.RedisPassword, c.RedisHost, c.RedisPort)
	}
	return fmt.Sprintf("redis://%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
