package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	models "github.com/romashorodok/stream-platform/services/stream/internal/storage/schema/postgres/public/model"
	. "github.com/romashorodok/stream-platform/services/stream/internal/storage/schema/postgres/public/table"

	"github.com/go-jet/jet/v2/qrm"
	"go.uber.org/fx"
)

type StreamEgressRepository struct {
	db *sql.DB
}

type StreamEgress struct {
	Type           string
	ActiveStreamID uuid.UUID
}

func (r *StreamEgressRepository) InsertManyActiveStreamEgresses(q qrm.Queryable, ctx context.Context, egresses ...StreamEgress) ([]models.ActiveStreamEgresses, error) {
	var models []models.ActiveStreamEgresses

	if q == nil {
		q = r.db
	}

	err := ActiveStreamEgresses.INSERT(
		ActiveStreamEgresses.Type,
		ActiveStreamEgresses.ActiveStreamID,
	).MODELS(&egresses).
		RETURNING(ActiveStreamEgresses.AllColumns).
		QueryContext(ctx, q, &models)

	if err != nil {
		return nil, err
	}

	return models, err
}

type StreamEgressRepositroyParams struct {
	fx.In

	DB *sql.DB
}

func NewStreamEgressRepository(params StreamEgressRepositroyParams) *StreamEgressRepository {
	return &StreamEgressRepository{db: params.DB}
}
