package portrange

import (
	"errors"
	"fmt"
	"sync"
)

type PortState struct {
	port       Port
	Identifier string
}

func (s *PortState) Port() Port {
	return s.port
}

type Port uint16

type PortRange struct {
	portMin Port
	portMax Port

	busyPortRange map[Port]*PortState
	mx            sync.Mutex
}

func (r *PortRange) GetPort() *PortState {
	r.mx.Lock()
	defer r.mx.Unlock()

	for i := Port(r.portMin); i <= r.portMax; i++ {
		_, exist := r.busyPortRange[i]

		if !exist {
			r.busyPortRange[i] = &PortState{port: i}
			return r.busyPortRange[i]
		}
	}

	return nil
}

func (r *PortRange) PortBack(port Port) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if err := validatePort(port); err != nil {
		return err
	}

	portState, exist := r.busyPortRange[port]
	if exist {
		if portState.port != port {
			return fmt.Errorf("different ports found when port back it. portState.port: %d. port: %d", portState.port, port)
		}

		delete(r.busyPortRange, port)
		return nil
	}

	return fmt.Errorf("unable delete port. Port: %d. portState: %p", port, portState)
}

const (
	MinPortBorder Port = 20000
	MaxPortBorder Port = 24999
)

func validatePort(port Port) error {
	if port == 0 {
		return errors.New("Port range is 0")
	}
	if port > MaxPortBorder {
		return fmt.Errorf("Port %d must be less than %d", port, MaxPortBorder)
	}
	if port < MinPortBorder {
		return fmt.Errorf("Port %d must be larger than %d", port, MinPortBorder)
	}
	return nil
}

func NewPortRange(portMin, portMax Port) (*PortRange, error) {
	if portMin == 0 && portMax == 0 {
		return nil, errors.New("Port range is 0")
	}
	if portMax-portMin == 0 {
		return nil, errors.New("Port range is 0")
	}
	if portMax > MaxPortBorder || portMax < MinPortBorder {
		return nil, fmt.Errorf("Max port must be less then %d and large then %d. portMax is %d", MaxPortBorder, MinPortBorder, portMax)
	}
	if portMax > MaxPortBorder || portMax < MinPortBorder {
		return nil, fmt.Errorf("Max port must be less then %d and large then %d. portMax is %d", MaxPortBorder, MinPortBorder, portMax)
	}
	if portMin > MaxPortBorder || portMin < MinPortBorder {
		return nil, fmt.Errorf("Max port must be less then %d and large then %d. portMin is %d", MaxPortBorder, MinPortBorder, portMin)
	}
	if portMin > portMax {
		return nil, errors.New("portMin must be less then portMax")
	}

	return &PortRange{
		portMin:       portMin,
		portMax:       portMax,
		busyPortRange: make(map[Port]*PortState),
	}, nil
}
