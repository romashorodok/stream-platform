package service

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
)

type NatsConfig struct {
	Port string
	Host string
}

func (c *NatsConfig) GetUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.Host, c.Port)
}

func NewNatsConfig() *NatsConfig {
	return &NatsConfig{
		Host: envutils.Env(variables.NATS_HOST, variables.NATS_HOST_DEFAULT),
		Port: envutils.Env(variables.NATS_PORT, variables.NATS_PORT_DEFAULT),
	}
}

func NewNatsConnection(config *NatsConfig) (*nats.Conn, error) {
	return nats.Connect(config.GetUrl(), nats.Timeout(time.Second*5), nats.RetryOnFailedConnect(true))
}

var NatsModule = fx.Module("nats",
	// fx.Provide(
	// 	NewNatsConfig,
	// 	NewNatsConnection,
	// ),
)
