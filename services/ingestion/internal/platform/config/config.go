package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	AppPort     string
	LogLevel    string
	Environment string

	KafkaBrokers          []string
	KafkaTopic            string
	KafkaConsumerMaxRetry int
	KafkaConsumerWorkers  int

	DatabaseURL string
	TableName   string

	OllamaPort string
	EmbedModel string

	QdrantCollection string
	QdrantHost       string
	QdrantGRPCPort   int
}

func Load() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("INFO: No .env file found, reading from system environment")
	}

	return &AppConfig{
		AppPort:     getEnvOrDefault("APP_PORT", ":8080"),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
		Environment: getEnvOrDefault("ENVIRONMENT", "prod"),

		KafkaBrokers:          getEnvSliceOrDefault("KAFKA_BROKERS", "localhost:19092"),
		KafkaTopic:            getEnvOrDefault("KAFKA_TOPIC", "document-events"),
		KafkaConsumerMaxRetry: getEnvOrDefaultInt("KAFKA_CONSUMER_MAX_RETRY", 3),
		KafkaConsumerWorkers:  getEnvOrDefaultInt("KAFKA_CONSUMER_WORKERS", 16),

		DatabaseURL: getEnvOrDefault("DATABASE_URL", "postgres://postgres:password@localhost:5432/document"),
		TableName:   getEnvOrDefault("DB_NAME", "document"),

		OllamaPort: getEnvOrDefault("OLLAMA_HOST", "localhost:11434"),
		EmbedModel: getEnvOrDefault("EMBED_MODEL", "nomic-embed-text"),

		QdrantCollection: getEnvOrDefault("QDRANT_COLLECTION", "document"),
		QdrantHost:       getEnvOrDefault("QDRANT_HOST", "localhost"),
		QdrantGRPCPort:   getEnvOrDefaultInt("QDRANT_GRPC_PORT", 6334),
	}, nil

}

func getEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	return value
}

func getEnvOrDefaultInt(key string, fallback int) int {
	valueStr := os.Getenv(key)

	if valueStr == "" {
		return fallback
	}

	valueInt, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}

	return valueInt
}

func getEnvSliceOrDefault(key, fallback string) []string {
	value := getEnvOrDefault(key, fallback)
	return strings.Split(value, ",")
}
