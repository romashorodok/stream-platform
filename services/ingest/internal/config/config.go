package config

import (
	"log"
	"strconv"

	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/variables"
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
			Enable:   parseBool(envutils.Env(variables.TURN_ENABLE, variables.TURN_ENABLE_DEFAULT)),
			URL:      envutils.Env(variables.TURN_URL, variables.TURN_URL_DEFAULT),
			User:     envutils.Env(variables.TURN_USERNAME, variables.TURN_USERNAME_DEFAULT),
			Password: envutils.Env(variables.TURN_PASSWORD, variables.TURN_PASSWORD_DEFAULT),
		},
	}
}
