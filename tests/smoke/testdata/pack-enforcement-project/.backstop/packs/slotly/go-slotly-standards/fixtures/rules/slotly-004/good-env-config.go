package config

import "os"

// Config demonstrates the correct pattern: all secrets from environment.
type Config struct {
	SlackClientSecret  string
	SlackSigningSecret string
	EncryptionKey      string
	OpenAIAPIKey       string
}

func Load() *Config {
	return &Config{
		SlackClientSecret:  os.Getenv("SLACK_CLIENT_SECRET"),
		SlackSigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		EncryptionKey:      os.Getenv("ENCRYPTION_KEY"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
	}
}
