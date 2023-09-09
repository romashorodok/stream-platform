package wspeer

import (
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type WebsocketProtobuf[F protoreflect.ProtoMessage] struct {
	Type string `json:"type"`
	Data string `json:"data"`

	message F
}

func (p *WebsocketProtobuf[F]) Marshal() error {
	pjson := protojson.Format(p.message)

	msgType := p.message.ProtoReflect().Descriptor().FullName()
	if msgType == "" {
		return errors.New("empty descriptor name")
	}

	p.Type = string(msgType)
	p.Data = pjson

	return nil
}

func NewWebsocketProtobuf[F protoreflect.ProtoMessage](msg F) *WebsocketProtobuf[F] {
	return &WebsocketProtobuf[F]{
		message: msg,
	}
}
