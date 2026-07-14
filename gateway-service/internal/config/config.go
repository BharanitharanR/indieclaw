package config

import (
	"os"
)

type Config struct {
	GRPCPort           string
	PlannerEndpoint    string
	PrimaryLLMKey      string
	PrimaryLLMEndpoint string
}

func Load() *Config {
	return &Config{
		GRPCPort:           getEnv("GRPC_PORT", "50051"),
		PlannerEndpoint:    getEnv("PLANNER_ENDPOINT", "http://localhost:8081/v1/plan"),
		PrimaryLLMKey:      getEnv("PRIMARY_LLM_KEY", ""),
		PrimaryLLMEndpoint: getEnv("PRIMARY_LLM_ENDPOINT", "https://api.openai.com/v1/chat/completions"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
