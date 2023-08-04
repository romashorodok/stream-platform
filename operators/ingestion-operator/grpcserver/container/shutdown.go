package container

import (
	shutdwn "github.com/romashorodok/stream-platform/pkg/shutdown"
)

var shutdown = shutdwn.NewShutdown()

func WithShutdown() *shutdwn.Shutdown {
	return shutdown
}
