package portrange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPortRange_Creating(t *testing.T) {
	assert := assert.New(t)

	_, err := NewPortRange(0, 0)
	assert.NotNil(err, err)

	_, err = NewPortRange(1, 1)
	assert.NotNil(err, err)

	_, err = NewPortRange(200, 1000)
	assert.NotNil(err, err)

	_, err = NewPortRange(100, 90)
	assert.NotNil(err, err)

	_, err = NewPortRange(MinPortBorder-1, 90)
	assert.NotNil(err, err)

	_, err = NewPortRange(MinPortBorder, MaxPortBorder+1)
	assert.NotNil(err, err)

	_, err = NewPortRange(MinPortBorder, MaxPortBorder)
	assert.Nil(err, err)
}

func TestGetPort_Getting(t *testing.T) {
	assert := assert.New(t)

	r, _ := NewPortRange(MinPortBorder, MaxPortBorder)

	state := r.GetPort()
	assert.Equal(MinPortBorder, state.port, "Invalid port must")

	state = r.GetPort()
	assert.Equal(MinPortBorder+1, state.port, "Invalid port must")

	err := r.PortBack(state.port)
	assert.Nil(err, err)

	err = r.PortBack(state.port)
	assert.NotNil(err, err)

	err = r.PortBack(state.port - 1)
	assert.Nil(err, err)

	err = r.PortBack(state.port - 1)
	assert.NotNil(err, err)

	state = r.GetPort()
	assert.Equal(MinPortBorder, state.port)

	state = r.GetPort()
	assert.Equal(MinPortBorder+1, state.port)
}

func TestGetPort_Fullports(t *testing.T) {
	assert := assert.New(t)

	r, _ := NewPortRange(MinPortBorder, MaxPortBorder)

	for i := Port(MinPortBorder); i <= Port(MaxPortBorder); i++ {
		_ = r.GetPort()
	}

	var capacity Port = r.portMax - r.portMin + 1
	assert.Equal(capacity, Port(len(r.busyPortRange)))

	state := r.GetPort()
	assert.Nil(state, "full port container must be nil")

	state = r.GetPort()
	assert.Nil(state, "full port container must be nil")

	err := r.PortBack(MaxPortBorder)
	assert.Nil(err, "last port back error", err)

	state = r.GetPort()
	assert.Equal(MaxPortBorder, state.port, "Must be last port")

	var center Port = (r.portMin + r.portMax) / 2

	err = r.PortBack(center)
	assert.Nil(err, "center port back must be nil", err)

	state = r.GetPort()
	assert.Equal(center, state.port, "Must be center port")

	state = r.GetPort()
	assert.Nil(state, "Must be nil after delete center and get it back")
}
