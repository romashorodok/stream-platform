package openapi

import (
	"encoding/json"
	"regexp"
	"strings"
)

type OpenAPIError struct {
	Message string      `json:"message"`
	Value   interface{} `json:"value"`
	Schema  interface{} `json:"schema"`
}

const multipleErrorPattern = `^(.*?)(?:\nSchema:\n((?:.|\n)+?))?\nValue:\n((?:.|\n)+?)(?:\n.\| |$)`

type OpenAPIMultipleErrorHandler struct {
	container []OpenAPIError
}

func (hand *OpenAPIMultipleErrorHandler) Parse(message string) {
	regex := regexp.MustCompile(multipleErrorPattern)
	match := regex.FindAllStringSubmatch(message, -1)

	for _, tokens := range match {

		openAPIError := OpenAPIError{
			Message: tokens[1],
		}

		var value interface{}
		if err := json.Unmarshal([]byte(tokens[3]), &value); err == nil {
			openAPIError.Value = value
		}

		var schema interface{}
		if err := json.Unmarshal([]byte(tokens[2]), &schema); err == nil {
			openAPIError.Schema = schema
		} else {
			openAPIError.Schema = tokens[2]
		}

		hand.container = append(hand.container, openAPIError)

		message = strings.Replace(message, tokens[0], "", 1)

		hand.Parse(message)
	}
}

func (hand *OpenAPIMultipleErrorHandler) GetOpenAPIErrors() []OpenAPIError {
	return hand.container
}

func NewOpenAPIMultipleErrorHandler() *OpenAPIMultipleErrorHandler {
	return &OpenAPIMultipleErrorHandler{}
}
