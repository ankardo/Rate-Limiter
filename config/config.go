package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	RedisAddr        string
	RedisPassword    string
	MaxRequests      int
	TokenMaxRequests int
	BlockDuration    int
	TTLExpiration    int
}

func LoadConfig(envPath string) Config {
	err := godotenv.Load(envPath)
	if err != nil {
		log.Println("No .env file found, using environment variables", envPath)
	}

	maxRequests, _ := strconv.Atoi(getEnv("MAX_REQUESTS_PER_SECOND", "5"))
	tokenMaxRequests, _ := strconv.Atoi(getEnv("TOKEN_MAX_REQUESTS", "10"))
	blockDuration, _ := strconv.Atoi(getEnv("BLOCK_DURATION_SECONDS", "60"))
	ttlExpiration, _ := strconv.Atoi(getEnv("TTL_EXPIRATION_SECONDS", "60"))

	return Config{
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		MaxRequests:      maxRequests,
		TokenMaxRequests: tokenMaxRequests,
		BlockDuration:    blockDuration,
		TTLExpiration:    ttlExpiration,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
