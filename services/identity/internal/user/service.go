package user

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/romashorodok/stream-platform/services/identity/internal/security"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/privatekey"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/refreshtoken"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/user"
	"github.com/romashorodok/stream-platform/services/identity/pkg/tokenutils"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"
)

var DEFAULT_CLAIMS = []string{"user", "broadcaster"}

type UserServiceImpl struct {
	privateKeyRepository   *privatekey.PrivateKeyRepository
	userRepository         *user.UserRepository
	refreshTokneRepository *refreshtoken.RefreshTokenRepository
	securityService        *security.SecurityServiceImpl
	db                     *sql.DB
}

type userTokensResult struct {
	AccessToken  string
	RefreshToken http.Cookie
}

type userInfo struct {
	userID   uuid.UUID
	username string
	claims   []string
}

func (s *UserServiceImpl) userGenerateAccessToken(privateKeyJws, kid string, user userInfo) (string, error) {

	accessToken, err := s.securityService.CreateUserAccessToken(security.CreateUserAccessTokenParams{
		PrivateKeyJwsMessage: privateKeyJws,
		Kid:                  kid,
		Username:             user.username,
		UserID:               user.userID,
		Claims:               user.claims,
	})
	if err != nil {
		return "", fmt.Errorf("Unable generate access token. Error: %s", err)
	}

	return accessToken, err
}

func (s *UserServiceImpl) userGenerateRefreshToken(tx *sql.Tx, ctx context.Context, privateKeyJws string, kid uuid.UUID, user userInfo) (string, *time.Time, error) {

	refreshToken, expiresAt, err := s.securityService.CreateRefreshToken(security.CreateRefreshTokenParams{
		PrivateKeyJwsMessage: privateKeyJws,
		Kid:                  kid.String(),
		Username:             user.username,
		UserID:               user.userID,
		Claims:               user.claims,
	})

	if err != nil {
		return "", nil, fmt.Errorf("Unable generate refresh token. Error: %s", err)
	}

	refreshTokenStored, err := s.refreshTokneRepository.InsertRefreshToken(tx, ctx, kid, refreshToken, *expiresAt)
	if err != nil {
		return "", nil, fmt.Errorf("Unable store refresh token. Error: %s", err)
	}

	if err = s.userRepository.AttachRefreshToken(tx, ctx, user.userID, refreshTokenStored.ID); err != nil {
		return "", nil, fmt.Errorf("Unable store associated refresh token with user. Error.  Error: %s", err)
	}

	return refreshTokenStored.Plaintext, expiresAt, nil
}

func (s *UserServiceImpl) userGenerateTokens(tx *sql.Tx, ctx context.Context, user userInfo) (*userTokensResult, error) {

	privateKeyJws, err := tokenutils.CreatePrivateKeyAsJwsMessage()
	if err != nil {
		return nil, fmt.Errorf("Unable create private key. Error: %s", err)
	}

	privateKey, err := s.privateKeyRepository.InsertPrivateKey(tx, ctx, privateKeyJws)
	if err != nil {
		return nil, fmt.Errorf("Unable save private key record. Error: %s", err)
	}

	if err = s.userRepository.AttachPrivateKey(tx, ctx, user.userID, privateKey.ID); err != nil {
		return nil, fmt.Errorf("Unable store associated private key with user. Error: %s", err)
	}

	kid := privateKey.ID.String()

	refreshToken, expiresAt, err := s.userGenerateRefreshToken(tx, ctx, privateKey.JwsMessage, privateKey.ID, user)
	if err != nil {
		return nil, fmt.Errorf("Unable generate refresh token. Error: %s", err)
	}

	accessToken, err := s.userGenerateAccessToken(privateKey.JwsMessage, kid, user)
	if err != nil {
		return nil, err
	}

	return &userTokensResult{
		AccessToken: accessToken,
		RefreshToken: http.Cookie{
			Name:     tokenutils.REFRESH_TOKEN_COOKIE_NAME,
			Value:    refreshToken,
			Expires:  *expiresAt,
			HttpOnly: true,
			Path:     "/",
		},
	}, nil
}

func (s *UserServiceImpl) userRefreshTokens(tx *sql.Tx, ctx context.Context, kid uuid.UUID) (*userTokensResult, error) {

	user, err := s.userRepository.FindUserByPrivateKey(kid)
	if err != nil {
		return nil, fmt.Errorf("Unable find user by private key. Error: %s", err)
	}

	pkey, err := s.securityService.GetPrivateKey(kid)
	if err != nil {
		return nil, fmt.Errorf("Unable find private key. Error: %s", err)
	}

	accessToken, err := s.securityService.CreateUserAccessToken(security.CreateUserAccessTokenParams{
		PrivateKeyJwsMessage: pkey.JwsMessage,
		Kid:                  kid.String(),
		Username:             user.Username,
		UserID:               user.ID,
		Claims:               DEFAULT_CLAIMS,
	})
	if err != nil {
		return nil, fmt.Errorf("Unable create access token. Error: %s", err)
	}

	if err = s.refreshTokneRepository.DeleteRefreshTokenByPrivateKey(tx, ctx, kid); err != nil {
		return nil, fmt.Errorf("Unable delete old access token. Error: %s", err)
	}

	refreshToke, expiresAt, err := s.userGenerateRefreshToken(
		tx,
		ctx,
		pkey.JwsMessage,
		kid,
		userInfo{
			userID:   user.ID,
			username: user.Username,
			claims:   DEFAULT_CLAIMS,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("Unable refresh token. Error: %s", err)
	}

	return &userTokensResult{
		AccessToken: accessToken,
		RefreshToken: http.Cookie{
			Name:     tokenutils.REFRESH_TOKEN_COOKIE_NAME,
			Value:    refreshToke,
			Expires:  *expiresAt,
			HttpOnly: true,
			Path:     "/",
		},
	}, nil
}

func (s *UserServiceImpl) RegisterUser(ctx context.Context, username, password string) (*userTokensResult, error, int) {

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("Unable hash password. Error: %s", err), http.StatusInternalServerError
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable start transaction, Error: %s", err), http.StatusInternalServerError
	}
	defer tx.Rollback()

	user, err := s.userRepository.InsertUser(tx, ctx, username, string(hashed))
	if err != nil {
		return nil, fmt.Errorf("Unable insert user record. Error: %s", err), http.StatusInternalServerError
	}

	result, err := s.userGenerateTokens(tx, ctx, userInfo{
		userID:   user.ID,
		username: user.Username,
		claims:   DEFAULT_CLAIMS,
	})
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	if err = tx.Commit(); err != nil {
		return nil, err, http.StatusInternalServerError
	}

	return result, nil, http.StatusCreated
}

func (s *UserServiceImpl) UserLogin(ctx context.Context, username, password string) (*userTokensResult, error, int) {

	user, err := s.userRepository.FindUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("Not found user. Error: %s", err), http.StatusNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("Different password. Error: %s", err), http.StatusUnprocessableEntity
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable start transaction. Error: %s", err), http.StatusInternalServerError
	}
	defer tx.Rollback()

	result, err := s.userGenerateTokens(tx, ctx, userInfo{
		userID:   user.ID,
		username: user.Username,
		claims:   DEFAULT_CLAIMS,
	})
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	if err = tx.Commit(); err != nil {
		return nil, err, http.StatusInternalServerError
	}

	return result, nil, http.StatusOK
}

func (s *UserServiceImpl) UserExchangeAccessToken(ctx context.Context, rawRefreshToken string) (*userTokensResult, error) {

	_, err := s.securityService.ValidateToken(rawRefreshToken)
	if err != nil {
		return nil, err
	}

	kid, err := s.securityService.GetUserKID(rawRefreshToken)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable start transaction. Error: %s", err)
	}
	defer tx.Rollback()

	tokens, err := s.userRefreshTokens(tx, ctx, *kid)
	if err != nil {
		return nil, fmt.Errorf("Unable refresh tokens. Error: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *UserServiceImpl) UserDeleteRefreshToken(ctx context.Context, rawRefreshToken string) error {
	kid, err := s.securityService.GetUserKID(rawRefreshToken)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Unable start transaction. Error: %s", err)
	}
	defer tx.Rollback()

	err = s.refreshTokneRepository.DeleteRefreshTokenByPrivateKey(tx, ctx, *kid)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("Unable delete refresh token. Error: %s", err)
	}

	err = s.privateKeyRepository.DeletePrivateKey(tx, ctx, *kid)
	if err != nil {
		return fmt.Errorf("Unable delete private key. Error: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

type UserServiceParams struct {
	fx.In

	PrivateKeyRepository   *privatekey.PrivateKeyRepository
	UserRepository         *user.UserRepository
	SecurityService        *security.SecurityServiceImpl
	RefreshToeknRepository *refreshtoken.RefreshTokenRepository
	DB                     *sql.DB
}

func NewUserService(params UserServiceParams) *UserServiceImpl {
	return &UserServiceImpl{
		privateKeyRepository:   params.PrivateKeyRepository,
		userRepository:         params.UserRepository,
		securityService:        params.SecurityService,
		refreshTokneRepository: params.RefreshToeknRepository,
		db:                     params.DB,
	}
}
