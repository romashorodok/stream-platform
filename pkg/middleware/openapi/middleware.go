package openapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type ErrorHandler func(w http.ResponseWriter, message string, statusCode int)

type MultiErrorHandler func(openapi3.MultiError) (int, error)

type Options struct {
	Options               openapi3filter.Options
	ErrorHandler          ErrorHandler
	MultiErrorHandler     MultiErrorHandler
	SilenceServersWarning bool
}

type MultipleErrorResponse struct {
	Messages []OpenAPIError `json:"messages"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func validateRequest(req *http.Request, router routers.Router, options *Options) (int, interface{}) {
	route, pathParams, err := router.FindRoute(req)

	if err != nil {
		return http.StatusNotFound, &ErrorResponse{Message: err.Error()}
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	if options != nil {
		requestValidationInput.Options = &options.Options
	}

	if err := openapi3filter.ValidateRequest(req.Context(), requestValidationInput); err != nil {
		log.Println(err)

		switch e := err.(type) {
		case openapi3.MultiError:
			handler := NewOpenAPIMultipleErrorHandler()
			handler.Parse(e.Error())

			messages := handler.GetOpenAPIErrors()

			if messages == nil {
				return http.StatusBadRequest, &ErrorResponse{
					Message: err.Error(),
				}
			}

			return http.StatusBadRequest, &MultipleErrorResponse{
				Messages: messages,
			}

		case *openapi3filter.SecurityRequirementsError:
			return http.StatusUnauthorized, &ErrorResponse{Message: e.Error()}

		case *openapi3filter.RequestError:
			return http.StatusBadRequest, &ErrorResponse{Message: e.Error()}

		default:
			return http.StatusInternalServerError, &ErrorResponse{Message: e.Error()}
		}
	}

	return 0, nil
}

func emptyBearerToken(header string) error {
	prefix := "Bearer "

	if header == "" && !strings.HasPrefix(header, prefix) {
		return errors.New("empty bearer token")
	}

	token := strings.TrimPrefix(header, prefix)

	if token == "" {
		return errors.New("bad token format")
	}

	return nil
}

func NewOpenAPIRequestMiddleware(spec *openapi3.T, options *Options) func(http.Handler) http.Handler {
	router, err := gorillamux.NewRouter(spec)

	if err != nil {
		log.Panicf("Unable start openapi router %s", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// if err := emptyBearerToken(r.Header.Get("Authorization")); err != nil {
			// 	w.Header().Set("Content-Type", "application/json")
			// 	w.WriteHeader(http.StatusUnauthorized)
			// 	json.NewEncoder(w).Encode(&ErrorResponse{Message: err.Error()})

			// 	return
			// }

			if status, errorResponse := validateRequest(r, router, options); errorResponse != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(errorResponse)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
