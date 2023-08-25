package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/privatekey"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/refreshtoken"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/user"
	"github.com/romashorodok/stream-platform/services/identity/pkg/tokenutils"

	"go.uber.org/fx"
)

type SecurityServiceImpl struct {
	refreshTokenRepository *refreshtoken.RefreshTokenRepository
	privateKeyRepository   *privatekey.PrivateKeyRepository
	userRepository         *user.UserRepository
}

type CreateUserAccessTokenParams struct {
	PrivateKeyJwsMessage string
	Kid                  string
	Username             string
	Claims               []string
	UserID               uuid.UUID
}

func (s *SecurityServiceImpl) CreateUserAccessToken(params CreateUserAccessTokenParams) (string, error) {
	expiresAt := time.Now().Add(time.Minute * 1)

	token, err := s.CreateToken(params.Username, params.Claims, expiresAt)
	if err != nil {
		return "", fmt.Errorf("Unable create access token. Error: %s", err)
	}

	if err = token.Set("user:id", params.UserID); err != nil {
		return "", fmt.Errorf("Unable set `user:id` claim. Error: %s", err)
	}

	if err = token.Set("token:use", "access_token"); err != nil {
		return "", fmt.Errorf("unable set `token:use` claim. Error: %s", err)
	}

	headers := jws.NewHeaders()
	headers.Set(jws.KeyIDKey, params.Kid)

	signed, err := s.signToken(params.PrivateKeyJwsMessage, headers, token)
	if err != nil {
		return "", fmt.Errorf("Unable sign token. Error: %s", err)
	}

	return string(signed), err
}

type CreateRefreshTokenParams struct {
	PrivateKeyJwsMessage string
	Kid                  string
	Username             string
	Claims               []string
	UserID               uuid.UUID
}

func (s *SecurityServiceImpl) CreateRefreshToken(params CreateRefreshTokenParams) (string, *time.Time, error) {
	expiresAt := time.Now().AddDate(1, 0, 0)

	token, err := s.CreateToken(params.Username, params.Claims, expiresAt)
	if err != nil {
		return "", nil, fmt.Errorf("Unable create refresh token. Error: %s", err)
	}

	if err = token.Set("user:id", params.UserID); err != nil {
		return "", nil, fmt.Errorf("Unable set `user:id` claim. Error: %s", err)
	}

	if err = token.Set("token:use", "refresh_token"); err != nil {
		return "", nil, fmt.Errorf("unable set `token:use` claim. Error: %s", err)
	}

	headers := jws.NewHeaders()
	headers.Set(jws.KeyIDKey, params.Kid)

	signed, err := s.signToken(params.PrivateKeyJwsMessage, headers, token)
	if err != nil {
		return "", nil, fmt.Errorf("Unable sign refresh token. Error: %s", err)
	}

	return string(signed), &expiresAt, nil
}

func (s *SecurityServiceImpl) CreateToken(username string, claims []string, expiresAt time.Time) (jwt.Token, error) {
	return jwt.NewBuilder().
		Issuer("0.0.0.0").
		Audience(claims).
		Subject(username).
		Expiration(expiresAt).
		Build()
}

func (s *SecurityServiceImpl) signToken(signKey string, headers jws.Headers, token jwt.Token) ([]byte, error) {
	sign, err := jwk.ParseKey([]byte(signKey))

	if err != nil {
		return nil, fmt.Errorf("unable serialize jws json message as jwk.Token. Error: %s\n", err)
	}

	return jwt.Sign(token, jwt.WithKey(jwa.RS256, sign, jws.WithProtectedHeaders(headers)))
}

func (s *SecurityServiceImpl) getKeySet(kid string, privateKey jwk.Key) jwk.Set {
	keyset := jwk.NewSet()

	// TODO: Don't hardcode public keys
	pbkey, _ := jwk.PublicKeyOf(privateKey)
	_ = pbkey.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = pbkey.Set(jwk.KeyIDKey, kid)
	_ = keyset.AddKey(pbkey)

	return keyset
}

func (s *SecurityServiceImpl) getPrivateKey(kid uuid.UUID) (jwk.Key, error) {
	privateKeyRecord, err := s.privateKeyRepository.GetPrivateKeyById(kid)
	if err != nil {
		return nil, fmt.Errorf("Unable find private key record. Error: %s", err)
	}

	return tokenutils.JwkPrivateKey(privateKeyRecord.JwsMessage)
}

// NOTE: Should i validate the token ?
func (s *SecurityServiceImpl) GetPublciKeys(rawToken string) ([]byte, error) {
	plainToken := tokenutils.TrimTokenBearer(rawToken)

	if plainToken == "" {
		return nil, errors.New("empty token")
	}

	tokenJws, err := tokenutils.InsecureJwsMessage(plainToken)
	if err != nil {
		return nil, fmt.Errorf("Unable get cast plain token to jws message. Error: %s", err)
	}

	kid := tokenJws.Signatures()[0].ProtectedHeaders().KeyID()
	kidUUID, err := uuid.Parse(kid)

	privateKey, err := s.getPrivateKey(kidUUID)
	if err != nil {
		return nil, fmt.Errorf("Unable get private key. Error: %s", err)
	}

	keyset := s.getKeySet(kid, privateKey)

	return tokenutils.SerializeKeyset(keyset)
}

// Return valid token payload. May be decoded as json
func (s *SecurityServiceImpl) ValidateToken(rawToken string) ([]byte, error) {
	plainToken := tokenutils.TrimTokenBearer(rawToken)

	if plainToken == "" {
		return nil, errors.New("empty token")
	}

	tokenJws, err := tokenutils.InsecureJwsMessage(plainToken)
	if err != nil {
		return nil, fmt.Errorf("Unable get cast plain token to jws message. Error: %s", err)
	}

	kid := tokenJws.Signatures()[0].ProtectedHeaders().KeyID()
	kidUUID, err := uuid.Parse(kid)
	if err != nil {
		return nil, fmt.Errorf("Unable parse kid as uuid")
	}

	privateKey, err := s.getPrivateKey(kidUUID)
	if err != nil {
		return nil, fmt.Errorf("Unable get private key. Error: %s", err)
	}

	keyset := s.getKeySet(kid, privateKey)

	verifiedPayload, err := tokenutils.VerifyTokenByKeySet(keyset, plainToken)
	if err != nil {
		return nil, fmt.Errorf("Not verified token. Error: %s", err)
	}

	err = tokenutils.ValidateToken(string(verifiedPayload))
	if err != nil {
		return nil, fmt.Errorf("Invalid token. Error: %s", err)
	}

	return verifiedPayload, nil
}

func (s *SecurityServiceImpl) GetUserKID(rawToken string) (*uuid.UUID, error) {
	plainToken := tokenutils.TrimTokenBearer(rawToken)
	tokenJws, err := tokenutils.InsecureJwsMessage(plainToken)
	if err != nil {
		return nil, fmt.Errorf("Unable get cast plain token to jws message. Error: %s", err)
	}

	kid := tokenJws.Signatures()[0].ProtectedHeaders().KeyID()
	kidUUID, err := uuid.Parse(kid)
	if err != nil {
		return nil, fmt.Errorf("Unable parse kid as uuid")
	}

	return &kidUUID, nil
}

type getPrivateKeyResult struct {
	JwsMessage string
}

func (s *SecurityServiceImpl) GetPrivateKey(kid uuid.UUID) (*getPrivateKeyResult, error) {

	privateKeyRecord, err := s.privateKeyRepository.GetPrivateKeyById(kid)
	if err != nil {
		return nil, fmt.Errorf("Unable find private key record. Error: %s", err)
	}

	return &getPrivateKeyResult{JwsMessage: privateKeyRecord.JwsMessage}, nil
}

type SecurityServiceParams struct {
	fx.In

	RefreshTokenRepository *refreshtoken.RefreshTokenRepository
	PrivateKeyRepository   *privatekey.PrivateKeyRepository
	UserRepository         *user.UserRepository
}

func NewSecurityService(params SecurityServiceParams) *SecurityServiceImpl {
	return &SecurityServiceImpl{
		privateKeyRepository:   params.PrivateKeyRepository,
		refreshTokenRepository: params.RefreshTokenRepository,
		userRepository:         params.UserRepository,
	}
}
