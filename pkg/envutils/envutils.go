package envutils

import (
	"log"
	"os"
	"strconv"
)

func ParseUint16(value string) (*uint16, error) {
	result, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return nil, err
	}
	cast := uint16(result)
	return &cast, nil
}

func ParseBool(value string) (*bool, error) {
	result, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	cast := bool(result)
	return &cast, nil
}

func Env(variableName, defaultValue string) string {
	if variable := os.Getenv(variableName); variable != "" {
		log.Printf("[%s]: %s", variableName, variable)
		return variable
	}
	log.Printf("[%s_DEFAULT]: %s", variableName, defaultValue)
	return defaultValue
}
