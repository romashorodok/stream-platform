package envutils

import (
	"log"
	"os"
)

func Env(variableName, defaultValue string) string {
	if variable := os.Getenv(variableName); variable != "" {
		log.Printf("[%s]: %s", variableName, variable)
		return variable
	}
	return defaultValue
}
