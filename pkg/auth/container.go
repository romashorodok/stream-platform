package auth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

const TOKEN_CONTEXT_VALUE = "__token_payload"

type TokenPayload struct {
	Aud      []string  `json:"aud"`
	Exp      int       `json:"exp"`
	Iss      string    `json:"iss"`
	Sub      string    `json:"sub"` /* Username */
	TokenUse string    `json:"token:use"`
	UserID   uuid.UUID `json:"user:id"`
}

func WithTokenPayload(ctx context.Context) (*TokenPayload, error) {
	tokenPayload, ok := ctx.Value(TOKEN_CONTEXT_VALUE).([]byte)
	if !ok {
		return nil, errors.New("unable get token payload. Usere may not be logged in.")
	}

	var payload TokenPayload

	if err := json.Unmarshal(tokenPayload, &payload); err != nil {
		return nil, errors.New("unable deserialize token payload.")
	}

	return &payload, nil
}
