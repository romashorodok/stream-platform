package config

import (
	"log"
	"os"
	"strconv"
)

type Turn struct {
	Enable   bool
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

func parseBool(value string) bool {
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Panic(err)
	}
	return boolValue

}

func NewConfig() *Config {
	return &Config{
		Turn: Turn{
			Enable:   parseBool(env("TURN_ENABLE", "false")),
			URL:      env("TURN_URL", ""),
			User:     env("TURN_USER", ""),
			Password: env("TURN_PASSWORD", ""),
		},
	}
}
