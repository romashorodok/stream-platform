package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/schema/postgres/public/model"
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

type UpdateDeployedStatusByBroadcasterIdResult struct {
	ID            uuid.UUID
	BroadcasterID uuid.UUID
	Namespace     string
	Deployment    string
	Running       bool
	Deployed      bool
}

func (r *ActiveStreamRepository) UpdateDeployedStatusByBroadcasterId(q qrm.Queryable, ctx context.Context, broadcasterID uuid.UUID, deployed bool) (*UpdateDeployedStatusByBroadcasterIdResult, error) {
	model := model.ActiveStreams{Deployed: deployed}

	if q == nil {
		q = r.db
	}

	err := ActiveStreams.UPDATE(ActiveStreams.Deployed).
		MODEL(model).
		WHERE(ActiveStreams.BroadcasterID.EQ(UUID(broadcasterID))).
		RETURNING(
			ActiveStreams.ID,
			ActiveStreams.BroadcasterID,
			ActiveStreams.Namespace,
			ActiveStreams.Deployment,
			ActiveStreams.Running,
			ActiveStreams.Deployed,
		).
		QueryContext(ctx, q, &model)

	if err != nil {
		return nil, err
	}

	return &UpdateDeployedStatusByBroadcasterIdResult{
		ID:            model.ID,
		BroadcasterID: model.BroadcasterID,
		Namespace:     model.Namespace,
		Deployment:    model.Deployment,
		Running:       model.Running,
		Deployed:      model.Deployed,
	}, err
}

type GetActiveStreamByBroadcasterIdResponse struct {
	ID            uuid.UUID
	BroadcasterID uuid.UUID
	Namespace     string
	Deployment    string
	Running       bool
	Deployed      bool
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
		Running:       model.Running,
		Deployed:      model.Deployed,
	}, nil
}

type RunningActiveStreamEgress struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
}

type getAllRunningActiveStreamsQuery struct {
	ID       uuid.UUID `json:"active_stream_id"`
	Username string    `json:"username"`

	Egresses []string `json:"egresses"`
}

type RunningActiveStreams struct {
	ID       uuid.UUID `json:"active_stream_id"`
	Username string    `json:"username"`

	Egresses []RunningActiveStreamEgress `json:"egresses"`
}

// NOTE: to map result into struct alias must have name of the struct and path of field separated by dot
const getAllRunningActiveStreamsAlias = "get_all_running_active_streams_query"

func (r *ActiveStreamRepository) GetAllRunningActiveStreams(ctx context.Context) ([]RunningActiveStreams, error) {
	var result []RunningActiveStreams

	stmt := SELECT(
		ActiveStreams.ID.AS(fmt.Sprintf("%s.id", getAllRunningActiveStreamsAlias)),
		ActiveStreams.Username.AS(fmt.Sprintf("%s.username", getAllRunningActiveStreamsAlias)),
		Raw(
			"JSON_AGG(JSON_BUILD_OBJECT('id', active_stream_egresses.id, 'type', active_stream_egresses.type))",
		).AS("egresses"),
	).FROM(
		ActiveStreams.INNER_JOIN(ActiveStreamEgresses, ActiveStreams.ID.EQ(
			ActiveStreamEgresses.ActiveStreamID,
		)),
	).GROUP_BY(
		ActiveStreams.ID,
	)

	rows, err := stmt.Rows(ctx, r.db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var model getAllRunningActiveStreamsQuery
		var egresses []RunningActiveStreamEgress

		err := rows.Scan(&model)
		if err != nil {
			log.Println("[GetAllRunningActiveStreams]: Unable scan row. Err:", err)
			continue
		}

		for _, egress := range model.Egresses {
			if err = json.Unmarshal([]byte(egress), &egresses); err != nil {
				log.Println("Unable deserialize GetAllRunningActiveStreams.Egresses json. Err", err)
				continue
			}
		}

		result = append(result, RunningActiveStreams{
			ID:       model.ID,
			Username: model.Username,
			Egresses: egresses,
		})
	}

	return result, nil
}

type ActiveStreamRepositoryParams struct {
	fx.In

	DB *sql.DB
}

func NewActiveStreamRepository(params ActiveStreamRepositoryParams) *ActiveStreamRepository {
	repo := &ActiveStreamRepository{db: params.DB}

	return repo
}
