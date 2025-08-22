package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	AI       AIConfig
	ArXiv    ArXivConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type AIConfig struct {
	OpenAIAPIKey     string
	OpenAIModel      string
	AnthropicAPIKey  string
	AnthropicModel   string
	AnthropicBaseURL string
	Provider         string
}

type ArXivConfig struct {
	Category     string
	MaxResults   int
	DelaySeconds float64
	NumRetries   int
}

func Load() (*Config, error) {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "arxiv_db"),
		},
		AI: AIConfig{
			OpenAIAPIKey:     getEnv("OPENAI_API_KEY", ""),
			OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-4o"),
			AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", "sk-kt5Sm80ylG6TIX8lemm5Fry7YYNx8DqDansWuO4wBhqD5inR"),
			AnthropicModel:   getEnv("ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
			AnthropicBaseURL: getEnv("ANTHROPIC_BASE_URL", "https://api.openai-proxy.org/anthropic"),
			Provider:         getEnv("AI_PROVIDER", "claude"),
		},
		ArXiv: ArXivConfig{
			Category:     getEnv("ARXIV_CATEGORY", "cs.AI"),
			MaxResults:   getEnvAsInt("ARXIV_MAX_RESULTS", 100),
			DelaySeconds: getEnvAsFloat("ARXIV_DELAY_SECONDS", 3.0),
			NumRetries:   getEnvAsInt("ARXIV_NUM_RETRIES", 3),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}