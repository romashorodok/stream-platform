package user

import (
	"context"
	"database/sql"
	"sync"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	models "github.com/romashorodok/stream-platform/services/identity/internal/storage/schema/postgres/public/model"
	. "github.com/romashorodok/stream-platform/services/identity/internal/storage/schema/postgres/public/table"

	"go.uber.org/fx"
)

type UserRepository struct {
	db *sql.DB
}

type InsertUserResponse struct {
	ID       uuid.UUID
	Username string
}

func (r *UserRepository) InsertUser(q qrm.Queryable, ctx context.Context, username, password string) (*InsertUserResponse, error) {
	var model models.Users

	if q == nil {
		q = r.db
	}

	err := Users.INSERT(Users.Username, Users.Password).
		VALUES(username, password).
		RETURNING(Users.ID, Users.Username).
		QueryContext(ctx, q, &model)

	if err != nil {
		return nil, err
	}

	return &InsertUserResponse{
		ID:       model.ID,
		Username: model.Username,
	}, nil
}

type findUserByUsernameResult struct {
	Username string
	Password string
	ID       uuid.UUID
}

func (r *UserRepository) FindUserByUsername(username string) (*findUserByUsernameResult, error) {
	var user models.Users

	err := Users.SELECT(Users.AllColumns).WHERE(Users.Username.EQ(String(username))).Query(r.db, &user)
	if err != nil {
		return nil, err
	}

	return &findUserByUsernameResult{
		Username: user.Username,
		Password: user.Password,
		ID:       user.ID,
	}, nil
}

type findUserByPrivateKeyResult struct {
	ID       uuid.UUID `sql:"primary_key"`
	Username string
}

func (r *UserRepository) FindUserByPrivateKey(privateKeyID uuid.UUID) (*findUserByPrivateKeyResult, error) {
	var user models.Users
	var userPKeys models.UserPrivateKeys

	err := UserPrivateKeys.SELECT(UserPrivateKeys.UserID).
		WHERE(UserPrivateKeys.PrivateKeyID.EQ(UUID(privateKeyID))).
		Query(r.db, &userPKeys)
	if err != nil {
		return nil, err
	}

	err = Users.SELECT(Users.AllColumns).
		WHERE(Users.ID.EQ(UUID(userPKeys.UserID))).
		Query(r.db, &user)

	if err != nil {
		return nil, err
	}

	return &findUserByPrivateKeyResult{
		ID:       user.ID,
		Username: user.Username,
	}, nil
}

func (r *UserRepository) AttachPrivateKey(q qrm.Executable, ctx context.Context, user_id uuid.UUID, private_key_id uuid.UUID) error {

	if q == nil {
		q = r.db
	}

	_, err := UserPrivateKeys.INSERT(
		UserPrivateKeys.UserID,
		UserPrivateKeys.PrivateKeyID,
	).VALUES(user_id, private_key_id).
		ExecContext(ctx, q)

	return err
}

func (r *UserRepository) AttachRefreshToken(q qrm.Executable, ctx context.Context, user_id uuid.UUID, refreshTokenId uuid.UUID) error {

	if q == nil {
		q = r.db
	}

	_, err := UserRefreshTokens.INSERT(
		UserRefreshTokens.UserID,
		UserRefreshTokens.RefreshTokenID,
	).
		VALUES(user_id, refreshTokenId).
		ExecContext(ctx, q)

	return err
}

type UserRepositoryParams struct {
	fx.In

	DB *sql.DB
}

var once sync.Once
var repo *UserRepository

func NewUserRepository(params UserRepositoryParams) *UserRepository {
	once.Do(func() {
		repo = &UserRepository{db: params.DB}
	})

	return repo
}
