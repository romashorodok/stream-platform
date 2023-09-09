package tokenutils

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const REFRESH_TOKEN_COOKIE_NAME = "_refresh_token"

// Create RS256 private key as jws message
func CreatePrivateKeyAsJwsMessage() (string, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("unable generate rsa private key. Error: %s\n", err)
	}

	key, err := jwk.FromRaw(pk)
	if err != nil {
		return "", fmt.Errorf("unable cast private key to jwk key. Error: %s\n", err)
	}

	proxy := struct{ Key jwk.Key }{Key: key}

	jsonBytes, err := json.Marshal(proxy.Key)
	if err != nil {
		return "", fmt.Errorf("unable serialize jwk key as json message. Error: %s\n", err)
	}

	return string(jsonBytes), nil
}

// Get jwt.Token from signed string. Need for getting token without verification and validation.
func InsecureToken(plainToken string) (jwt.Token, error) {
	return jwt.Parse([]byte(plainToken), jwt.WithVerify(false), jwt.WithValidate(false))
}

// Get jws.Message from signed string. Need for getting headers and signature from token without verification and validation.
func InsecureJwsMessage(plainToken string) (*jws.Message, error) {
	return jws.Parse([]byte(plainToken))
}

// Get jwk.Key from jws message string. Need for restoring RS256 private key from text.
func JwkPrivateKey(jwsMessage string) (jwk.Key, error) {
	key, err := jwk.ParseKey([]byte(jwsMessage))

	if err != nil {
		return nil, fmt.Errorf("unable generate rsa private key. Error: %s\n", err)
	}

	return key, nil
}

func TrimTokenBearer(plainToken string) string {
	return strings.TrimPrefix(plainToken, "Bearer ")
}

// Verify token signature by keyset. Key in set must match with token kid to be valid
// Returns verified token payload message
func VerifyTokenByKeySet(keyset jwk.Set, plainToken string) ([]byte, error) {
	return jws.Verify([]byte(plainToken), jws.WithKeySet(keyset, jws.WithRequireKid(true)))
}

// Check token payload. As example expire date, etc...
// Currently it's check only exp date
func ValidateToken(plainToken string) error {
	token, err := InsecureToken(plainToken)
	if err != nil {
		return fmt.Errorf("unable cast plain token to jwt.Token to validate it. Error: %s", err)
	}

	return jwt.Validate(token)
}

// Transform keyset into json
func SerializeKeyset(keyset jwk.Set) ([]byte, error) {
	proxy := struct{ Keys jwk.Set }{Keys: keyset}

	jsonKeyset, err := json.Marshal(proxy.Keys)
	if err != nil {
		return nil, fmt.Errorf("unable serialize jwk key as json message. Error: %s\n", err)
	}

	return jsonKeyset, nil
}
