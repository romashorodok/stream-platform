package refreshtoken

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/fx"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	models "github.com/romashorodok/stream-platform/services/identity/internal/storage/schema/postgres/public/model"
	. "github.com/romashorodok/stream-platform/services/identity/internal/storage/schema/postgres/public/table"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

type insertRefreshTokenResult struct {
	Plaintext string
	ID        uuid.UUID
}

func (r *RefreshTokenRepository) InsertRefreshToken(q qrm.Queryable, ctx context.Context, privateKeyID uuid.UUID, plaintextToken string, expiresAt time.Time) (*insertRefreshTokenResult, error) {
	var model models.RefreshTokens

	if q == nil {
		q = r.db
	}

	err := RefreshTokens.INSERT(RefreshTokens.PrivateKeyID, RefreshTokens.Plaintext, RefreshTokens.ExpiresAt).
		VALUES(privateKeyID, plaintextToken, expiresAt).
		RETURNING(RefreshTokens.ID, RefreshTokens.Plaintext).
		QueryContext(ctx, q, &model)

	if err != nil {
		return nil, err
	}

	return &insertRefreshTokenResult{
		Plaintext: model.Plaintext,
		ID:        model.ID,
	}, nil
}

func (r *RefreshTokenRepository) DeleteRefreshTokenByPrivateKey(q qrm.Executable, ctx context.Context, privateKeyID uuid.UUID) error {

	if q == nil {
		q = r.db
	}

	_, err := RefreshTokens.DELETE().WHERE(RefreshTokens.PrivateKeyID.EQ(UUID(privateKeyID))).ExecContext(ctx, q)
	if err != nil {
		return err
	}

	return nil
}

type RefreshTokenRepositoryParams struct {
	fx.In

	DB *sql.DB
}

func NewRefreshTokenRepository(params RefreshTokenRepositoryParams) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: params.DB,
	}
}
