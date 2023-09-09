package wspeer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	readDeadline  = 5 * time.Second
	writeDeadline = 5 * time.Second

	pingPeriod = 5 * time.Second
	pongWait   = 10 * time.Second
)

type WebsocketPeer struct {
	ws     *websocket.Conn
	mutex  sync.Mutex
	o      sync.Once
	msg    chan WebsocketMessage
	cancel context.CancelFunc
	ctx    context.Context
}

func (p *WebsocketPeer) Start() {

	p.o.Do(func() {

		go p.launch()
	})
}

func (p *WebsocketPeer) onError(err error) {
	log.Println(err)

	if _, ok := err.(*websocket.CloseError); ok {
		p.Close()
		return
	}

	if errors.Is(err, websocket.ErrCloseSent) {
		p.Close()
		return
	}
}

func (p *WebsocketPeer) launch() {

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		p.recv()
		p.cancel()
	}()

	wg.Wait()
}

func (c *WebsocketPeer) ping(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Write(websocket.PingMessage, nil)
		case <-ctx.Done():
			return
		}
	}
}

func (c *WebsocketPeer) recv() {
	for {
		select {
		case <-c.Done():
			return
		default:
			_, payload, err := c.ws.ReadMessage()

			if err != nil {
				c.onError(err)
				return
			}

			c.msg <- WebsocketMessage{Data: payload}
		}
	}
}

func (p *WebsocketPeer) Write(messageType int, data []byte) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if err := p.ws.WriteMessage(messageType, data); err != nil {
		p.onError(err)
	}
}

func (p *WebsocketPeer) WriteProtobuf(msg protoreflect.ProtoMessage) error {
	wsproto := NewWebsocketProtobuf(msg)
	if err := wsproto.Marshal(); err != nil {
		return err
	}

	w, err := p.ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	err1 := json.NewEncoder(w).Encode(wsproto)
	err2 := w.Close()
	if err1 != nil {
		return err1
	}

	return err2
}

func (p *WebsocketPeer) Recv() chan WebsocketMessage {
	return p.msg
}

func (p *WebsocketPeer) Close() {
	p.cancel()
	p.ws.Close()
}

func (p *WebsocketPeer) Done() <-chan struct{} {
	return p.ctx.Done()
}

type WebsocketMessage struct {
	Data []byte
}

func NewWebsocketPeer(ws *websocket.Conn, ctx context.Context) *WebsocketPeer {
	ctx, cancel := context.WithCancel(ctx)
	return &WebsocketPeer{
		ws:     ws,
		msg:    make(chan WebsocketMessage),
		cancel: cancel,
		ctx:    ctx,
	}
}
