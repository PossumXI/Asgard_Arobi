package db

import (
	"fmt"
	"os"
)

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

func LoadConfig() (*Config, error) {
	cfg := &Config{
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "asgard_secure_2026"),
		PostgresDB:       getEnv("POSTGRES_DB", "asgard"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		MongoHost:     getEnv("MONGO_HOST", "localhost"),
		MongoPort:     getEnv("MONGO_PORT", "27017"),
		MongoUser:     getEnv("MONGO_USER", "admin"),
		MongoPassword: getEnv("MONGO_PASSWORD", "asgard_mongo_2026"),
		MongoDB:       getEnv("MONGO_DB", "asgard"),

		NATSHost: getEnv("NATS_HOST", "localhost"),
		NATSPort: getEnv("NATS_PORT", "4222"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", "asgard_redis_2026"),
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
		"mongodb://%s:%s@%s:%s",
		c.MongoUser,
		c.MongoPassword,
		c.MongoHost,
		c.MongoPort,
	)
}

func (c *Config) NATSURI() string {
	return fmt.Sprintf("nats://%s:%s", c.NATSHost, c.NATSPort)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
