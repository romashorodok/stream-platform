package config

import "os"

type Turn struct {
	URL      string
	User     string
	Password string
}

type Config struct {
	Turn Turn
}

func env(key, defaultVariable string) string {
	if variable := os.Getenv(key); variable != "" {
		return variable
	}
	return defaultVariable
}

func NewConfig() *Config {
	return &Config{
		Turn: Turn{
			URL:      env("TURN_URL", ""),
			User:     env("TURN_USER", ""),
			Password: env("TURN_PASSWORD", ""),
		},
	}
}
