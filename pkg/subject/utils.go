package subject

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// https://docs.nats.io/nats-concepts/jetstream/headers
const NATS_MSG_ID = "Nats-Msg-Id"

func PublishProtobuf(conn *nats.Conn, subject string, message protoreflect.ProtoMessage) error {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("unable serialize message: Err: %s", err)
	}

	return conn.Publish(subject, bytes)
}

func JsPublishProtobuf(conn nats.JetStream, subject string, message protoreflect.ProtoMessage) (*nats.PubAck, error) {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("unable serialize message: Err: %s", err)
	}

	return conn.Publish(subject, bytes)
}

func JsPublishProtobufWithID(conn nats.JetStream, subject string, id string, message protoreflect.ProtoMessage) (*nats.PubAck, error) {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("unable serialize message: Err: %s", err)
	}

	msg := nats.NewMsg(subject)
	// TODO: If send same message with same id. It's will be like a duplicate and will dropped
	// msg.Header.Add(NATS_MSG_ID, id)
	msg.Data = bytes

	return conn.PublishMsg(msg)
}

func DeserializeProtobufMsg[F protoreflect.ProtoMessage](obj F, msg *nats.Msg) error {
	err := proto.Unmarshal(msg.Data, obj)
	if err != nil {
		return fmt.Errorf("unable deserialize message. Err: %s", err)
	}
	return nil
}
