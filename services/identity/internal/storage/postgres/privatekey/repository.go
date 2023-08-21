package privatekey

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

type PrivateKeyRepository struct {
	db *sql.DB
}

type createPrivateKeyResult struct {
	JwsMessage string
	ID         uuid.UUID
}

func (r *PrivateKeyRepository) InsertPrivateKey(q qrm.Queryable, ctx context.Context, jwsMessage string) (*createPrivateKeyResult, error) {
	var model models.PrivateKeys

	if q == nil {
		q = r.db
	}

	err := PrivateKeys.INSERT(PrivateKeys.JwsMessage).
		VALUES(jwsMessage).
		RETURNING(PrivateKeys.ID, PrivateKeys.JwsMessage).
		QueryContext(ctx, q, &model)

	if err != nil {
		return nil, err
	}

	return &createPrivateKeyResult{
		JwsMessage: model.JwsMessage,
		ID:         model.ID,
	}, nil
}

type getUserPrivateKeyResult struct {
	JwsMessage string
}

type getPrivateKeyById struct {
	JwsMessage string
}

func (r *PrivateKeyRepository) GetPrivateKeyById(id uuid.UUID) (*getPrivateKeyById, error) {
	var model models.PrivateKeys

	err := SELECT(PrivateKeys.JwsMessage).
		FROM(PrivateKeys).
		WHERE(PrivateKeys.ID.EQ(UUID(id))).
		Query(r.db, &model)

	if err != nil {
		return nil, err
	}

	return &getPrivateKeyById{
		JwsMessage: model.JwsMessage,
	}, nil
}

func (r *PrivateKeyRepository) DeletePrivateKey(q qrm.Executable, ctx context.Context, id uuid.UUID) error {
	if q == nil {
		q = r.db
	}

	_, err := PrivateKeys.DELETE().WHERE(PrivateKeys.ID.EQ(UUID(id))).ExecContext(ctx, q)
	if err != nil {
		return err
	}

	return nil
}

type PrivateKeyRepositoryParams struct {
	fx.In

	DB *sql.DB
}

var once sync.Once
var repo *PrivateKeyRepository

func NewPrivateKeyRepositroy(params PrivateKeyRepositoryParams) *PrivateKeyRepository {
	once.Do(func() {
		repo = &PrivateKeyRepository{db: params.DB}
	})

	return repo
}
