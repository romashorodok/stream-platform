package repository

import (
	"database/sql"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	models "github.com/romashorodok/stream-platform/services/stream/internal/storage/schema/postgres/public/model"
	. "github.com/romashorodok/stream-platform/services/stream/internal/storage/schema/postgres/public/table"
	"go.uber.org/fx"
)

type ActiveStreamRepository struct {
	db *sql.DB
}

type InsertActiveStreamResponse struct {
	ID            uuid.UUID
	BroadcasterID uuid.UUID
	Username      string
	Namespace     string
	Deployment    string
}

func (r *ActiveStreamRepository) InsertActiveStream(broadcasterID uuid.UUID, username, namespace, deployment string) (*InsertActiveStreamResponse, error) {

	var model models.ActiveStreams

	err := ActiveStreams.
		INSERT(
			ActiveStreams.BroadcasterID,
			ActiveStreams.Username,
			ActiveStreams.Namespace,
			ActiveStreams.Deployment,
		).
		VALUES(
			broadcasterID,
			username,
			namespace,
			deployment,
		).
		RETURNING(
			ActiveStreams.ID,
			ActiveStreams.BroadcasterID,
			ActiveStreams.Username,
			ActiveStreams.Namespace,
			ActiveStreams.Deployment,
		).
		Query(r.db, &model)

	if err != nil {
		return nil, err
	}

	return &InsertActiveStreamResponse{
		ID:            model.ID,
		BroadcasterID: model.BroadcasterID,
		Namespace:     model.Namespace,
		Deployment:    model.Deployment,
	}, nil
}

func (r *ActiveStreamRepository) DeleteActiveStreamByBroadcasterId(broadcasterID uuid.UUID) error {
	_, err := ActiveStreams.DELETE().
		WHERE(ActiveStreams.BroadcasterID.EQ(UUID(broadcasterID))).
		Exec(r.db)

	return err
}

type GetActiveStreamByBroadcasterIdResponse struct {
	ID            uuid.UUID
	BroadcasterID uuid.UUID
	Namespace     string
	Deployment    string
}

func (r *ActiveStreamRepository) GetActiveStreamByBroadcasterId(broadcasterID uuid.UUID) (*GetActiveStreamByBroadcasterIdResponse, error) {
	var model models.ActiveStreams
	err := SELECT(ActiveStreams.AllColumns).FROM(ActiveStreams.Table).
		WHERE(ActiveStreams.BroadcasterID.EQ(UUID(broadcasterID))).
		Query(r.db, &model)

	if err != nil {
		return nil, err
	}

	return &GetActiveStreamByBroadcasterIdResponse{
		ID:            model.ID,
		BroadcasterID: model.BroadcasterID,
		Namespace:     model.Namespace,
		Deployment:    model.Deployment,
	}, nil
}

type ActiveStreamRepositoryParams struct {
	fx.In

	DB *sql.DB
}

func NewActiveStreamRepository(params ActiveStreamRepositoryParams) *ActiveStreamRepository {
	repo := &ActiveStreamRepository{db: params.DB}

	return repo
}
