package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"github.com/romashorodok/stream-platform/services/identity/internal/security"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/privatekey"
	userrepo "github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/user"
	"github.com/romashorodok/stream-platform/services/identity/internal/user"
	"go.uber.org/fx"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../../gen/openapiv3/identity/v1alpha/service.openapi.yaml

type IdentityHandler struct {
	Unimplemented

	privKeyRepo *privatekey.PrivateKeyRepository
	userRepo    *userrepo.UserRepository
	userService *user.UserServiceImpl
	securitySvc *security.SecurityServiceImpl
	db          *sql.DB
}

var _ ServerInterface = (*IdentityHandler)(nil)

func (h *IdentityHandler) IdentityServiceSignIn(w http.ResponseWriter, r *http.Request) {
	var request SignInRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPreconditionFailed)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Unable deserialize request body. Error: %s",
				err.Error(),
			),
		})
		return
	}

	result, err, status := hand.userService.UserLogin(r.Context(), request.Username, request.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
		return
	}

	http.SetCookie(w, &result.RefreshToken)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		AccessToken string `json:"access_token"`
	}{
		AccessToken: result.AccessToken,
	})
}

func (h *IdentityHandler) IdentityServiceSignUp(w http.ResponseWriter, r *http.Request) {
	var request SignUpRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPreconditionFailed)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Unable deserialize request body. Error: %s",
				err.Error(),
			),
		})
		return
	}

	result, err, status := h.userService.RegisterUser(r.Context(), request.Username, request.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
		return
	}

	http.SetCookie(w, &result.RefreshToken)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		AccessToken string `json:"access_token"`
	}{
		AccessToken: result.AccessToken,
	})
}

func (h IdentityHandler) PublicKeyServicePublicKeyList(w http.ResponseWriter, r *http.Request) {
	plainToken := r.Header.Get("Authorization")

	if plainToken == "" {
		log.Println("Empty token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Empty token in authorization header",
			),
		})
		return
	}

	keyset, err := h.securitySvc.GetPublciKeys(plainToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(keyset)
}

func (h IdentityHandler) TokenRevocationServiceVerifyTokenRevocation(w http.ResponseWriter, r *http.Request) {
	plainToken := r.Header.Get("Authorization")

	if plainToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Empty token in authorization header",
			),
		})
		return
	}

	verified, err := h.securitySvc.ValidateToken(plainToken)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(verified)
}

const REFRESH_TOKEN_COOKIE_NAME = "_refresh_token"

func deleteRefreshTokenCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     REFRESH_TOKEN_COOKIE_NAME,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

func (h IdentityHandler) TokenServiceExchangeToken(w http.ResponseWriter, r *http.Request) {
	refreshTokenCookie, err := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
	if err != nil {
		deleteRefreshTokenCookie(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf("Error when getting cookie from request. Error: %s", err),
		})
		return
	}

	rawRefreshToken := refreshTokenCookie.Value
	if rawRefreshToken == "" {
		deleteRefreshTokenCookie(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf("Empty refresh token. Error: %s", err),
		})
		return
	}

	result, err := h.userService.UserExchangeAccessToken(r.Context(), rawRefreshToken)
	if err != nil {
		deleteRefreshTokenCookie(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf("Bad refresh token. Error: %s", err),
		})
		return
	}

	http.SetCookie(w, &result.RefreshToken)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		AccessToken string `json:"access_token"`
	}{
		AccessToken: result.AccessToken,
	})
}

func (h *IdentityHandler) IdentityServiceSignOut(w http.ResponseWriter, r *http.Request) {
	refreshTokenCookie, err := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
	if err != nil {
		deleteRefreshTokenCookie(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf("Error when getting cookie from request. Error: %s", err),
		})
		return
	}

	rawRefreshToken := refreshTokenCookie.Value
	if rawRefreshToken == "" {
		deleteRefreshTokenCookie(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf("Empty refresh token. Error: %s", err),
		})
		return
	}

	err = h.userService.UserDeleteRefreshToken(r.Context(), rawRefreshToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(ErrorResponse{Message: fmt.Sprintf("Something went wrong. Error: %s", err)})
		return
	}

	deleteRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
	w.Header().Set("Content-Type", "application/json")
}

type IdentityHandlerParams struct {
	fx.In

	PrivKeyRepo *privatekey.PrivateKeyRepository
	UserRepo    *userrepo.UserRepository
	SecuritySvc *security.SecurityServiceImpl
	UserService *user.UserServiceImpl
	DB          *sql.DB

	Lifecycle     fx.Lifecycle
	Router        *chi.Mux
	FilterOptions openapi3filter.Options
}

var once sync.Once
var hand *IdentityHandler

func NewIdentityHandler(params IdentityHandlerParams) *IdentityHandler {

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			once.Do(func() {
				hand = &IdentityHandler{
					privKeyRepo: params.PrivKeyRepo,
					userRepo:    params.UserRepo,
					securitySvc: params.SecuritySvc,
					userService: params.UserService,
					db:          params.DB,
				}
			})

			spec, err := GetSwagger()
			spec.Servers = nil

			if err != nil {
				return fmt.Errorf("unable get openapi spec. %s", err)
			}

			params.Router.Use(openapi.NewOpenAPIRequestMiddleware(spec, &openapi.Options{
				Options: params.FilterOptions,
			}))

			HandlerFromMux(hand, params.Router)

			return nil
		},
	})

	return hand
}
