package ingest

import (
	"context"
	"database/sql"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
)

type ingestStatusWorker struct {
	db                     *sql.DB
	conn                   *nats.Conn
	activeStreamRepository *repository.ActiveStreamRepository
	streamEgressRepository *repository.StreamEgressRepository
}

func (work *ingestStatusWorker) Start() {
	defer work.conn.Drain()

	work.conn.Subscribe(subject.IngestAnyUserDeployed, func(msg *nats.Msg) {
		var req subject.IngestDeployed

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		err := subject.DeserializeProtobufMsg(&req, msg)
		if err != nil {
			log.Printf("[Ingest Status Worker]: For %s failed to deserialize msg", msg.Data)
			return
		}

		broadcasterID, err := uuid.Parse(req.Meta.BroadcasterId)
		if err != nil {
			log.Println("unable parse broadcaster UUID")
			return
		}

		tx, err := work.db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("[Ingest Status Worker]: For %s failed to start tx", req.Meta.BroadcasterId)
			return
		}
		defer tx.Rollback()

		activeStream, err := work.activeStreamRepository.UpdateDeployedStatusByBroadcasterId(tx, ctx, broadcasterID, req.Deployed)
		if err != nil {
			log.Printf("[Ingest Status Worker]: For %s failed to update active stream status", req.Meta.BroadcasterId)
			return
		}

		var egressesDTO []repository.StreamEgress

		for _, egress := range req.Egresses {
			egressesDTO = append(egressesDTO, repository.StreamEgress{
				Type:           subjectpb.IngestEgressType_name[int32(egress.Type)],
				ActiveStreamID: activeStream.ID,
			})
		}

		_, err = work.streamEgressRepository.InsertManyActiveStreamEgresses(tx, ctx, egressesDTO...)
		if err != nil {
			log.Printf("[Ingest Status Worker]: For %s failed to start tx", req.Meta.BroadcasterId)
			return
		}

		_ = tx.Commit()
	})

	<-context.Background().Done()
}

type StartIngestWorkerParams struct {
	fx.In

	DB                     *sql.DB
	Conn                   *nats.Conn
	ActiveStreamRepository *repository.ActiveStreamRepository
	StreamEgressRepository *repository.StreamEgressRepository
}

func StartIngestStatusWorker(params StartIngestWorkerParams) {
	worker := ingestStatusWorker{
		db:                     params.DB,
		conn:                   params.Conn,
		activeStreamRepository: params.ActiveStreamRepository,
		streamEgressRepository: params.StreamEgressRepository,
	}

	go worker.Start()
}
