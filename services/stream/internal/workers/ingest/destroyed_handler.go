package ingest

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
)

type ingestDestroyedWorker struct {
	conn                   *nats.Conn
	activeStreamRepository *repository.ActiveStreamRepository
	js                     nats.JetStreamContext
	retry                  uint64
	retryInterval          time.Duration
}

func (work *ingestDestroyedWorker) handler(msg *nats.Msg) {
	var req subject.IngestDestroyed

	md, err := msg.Metadata()
	if err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable get metadata from subject: %s. Must be persistent jetstream message. Dropping msg from queue. Err: %s", msg.Subject, err)
		msg.Ack()
		return
	}

	if md.NumDelivered > work.retry {
		log.Printf("[Ingest Destroyed Worker] Reach retry limit for %s. Dropping msg from queue.", msg.Subject)
		msg.Ack()
		return
	}

	if err := subject.DeserializeProtobufMsg(&req, msg); err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable deserialize msg. Dropping msg from queue. Err: %s", err)
		msg.Ack()
		return
	}

	broadcasterID, err := uuid.Parse(req.Meta.BroadcasterId)
	if err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable get broadcasterID. Dropping msg from queue. Err: %s", err)
		msg.Ack()
		return
	}

	if err := work.activeStreamRepository.DeleteActiveStreamByBroadcasterId(broadcasterID); err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable delete active stream of %s. Retrying: %d. Err: %s", broadcasterID, md.NumDelivered, err)
		msg.NakWithDelay(work.retryInterval)
		return
	}

	if err := work.conn.Publish(subject.NewStreamDestroyedNotification(broadcasterID.String()), nil); err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable send notification for %s", broadcasterID)
		return
	}

	msg.Ack()
}

func (work *ingestDestroyedWorker) Start() {
	defer work.conn.Drain()

	if _, err := work.js.AddStream(subject.INGEST_DESTROYING_STREAM_CONFIG); err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable add stream %s. Err: %s", subject.INGEST_DESTROYING_STREAM, err)
	}
	defer work.js.DeleteStream(subject.INGEST_DESTROYING_STREAM)

	queueGroup := "ingest-destroyed-processor-1"
	subjectT := subject.IngestAnyUserDestroyed

	if _, err := work.js.AddConsumer(subject.INGEST_DESTROYING_STREAM, &nats.ConsumerConfig{
		Durable:        queueGroup,
		DeliverSubject: queueGroup,
		DeliverGroup:   queueGroup,
		AckWait:        time.Second,
		AckPolicy:      nats.AckExplicitPolicy,
		DeliverPolicy:  nats.DeliverLastPerSubjectPolicy,
		FilterSubject:  subjectT,
	}); err != nil {
		log.Printf("[Ingest Destroyed Worker] Unable add consumer %s. Err: %s", queueGroup, err)
	}
	defer work.js.DeleteConsumer(subject.INGEST_DESTROYING_STREAM, queueGroup)

	{
		opts := []nats.SubOpt{
			nats.Bind(subject.INGEST_DESTROYING_STREAM, queueGroup),
			nats.ManualAck(),
			nats.AckExplicit(),
			nats.DeliverLastPerSubject(),
		}

		if _, err := work.js.QueueSubscribe(subjectT, queueGroup, work.handler, opts...); err != nil {
			log.Printf("[Ingest Destroyed Worker] Catch error at subscribe queue %s. Err: %s", queueGroup, err)
		}
	}

	<-context.Background().Done()
}

type StartIngestDestroyedWorkerParams struct {
	fx.In

	Conn                   *nats.Conn
	ActiveStreamRepository *repository.ActiveStreamRepository
	JS                     nats.JetStreamContext
}

func StartIngestDestroyedWorker(params StartIngestDestroyedWorkerParams) {
	worker := ingestDestroyedWorker{
		js:                     params.JS,
		conn:                   params.Conn,
		activeStreamRepository: params.ActiveStreamRepository,
		retry:                  3,
		retryInterval:          time.Second * 30,
	}

	go worker.Start()
}
