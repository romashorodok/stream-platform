package ingest

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
)

type ingestStatusWorker struct {
	conn                   *nats.Conn
	activeStreamRepository *repository.ActiveStreamRepository
}

func (work *ingestStatusWorker) Start() {
	defer work.conn.Drain()

	work.conn.Subscribe(subject.IngestAnyUserDeployed, func(msg *nats.Msg) {
		var req subject.IngestDeployed

		err := subject.DeserializeProtobufMsg(&req, msg)
		if err != nil {
			log.Println(err)
		}

		broadcasterID, err := uuid.Parse(req.Meta.BroadcasterId)
		if err != nil {
			log.Println("unable parse broadcaster UUID")
			return
		}

		work.activeStreamRepository.UpdateDeployedStatusByBroadcasterId(
			broadcasterID,
			req.Deployed,
		)
	})

	<-context.Background().Done()
}

type StartIngestWorkerParams struct {
	fx.In

	Conn                   *nats.Conn
	ActiveStreamRepository *repository.ActiveStreamRepository
}

func StartIngestStatusWorker(params StartIngestWorkerParams) {
	worker := ingestStatusWorker{conn: params.Conn, activeStreamRepository: params.ActiveStreamRepository}

	go worker.Start()
}
