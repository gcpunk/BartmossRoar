package factory

import (
	"errors"
	"fmt"
	"sync"

	"github.com/erashaperavm/BartmossRoar/core/transport"
)

func NewNode(protocol transport.Proto, addrIfTcp string, localPort int, connectExistingIfTorHS bool) (*transport.Node, error) {
	switch protocol {
	case transport.TCP:
		trans, err := transport.NewTCP(addrIfTcp)
		if err != nil {
			return nil, err
		}
		return &transport.Node{
			Trans:    trans,
			Mu:       sync.Mutex{},
			Protocol: protocol,
		}, nil
	case transport.Tor:
		trans, err := transport.NewTor(localPort, connectExistingIfTorHS)
		if err != nil {
			return nil, err
		}
		return &transport.Node{
			Trans:    trans,
			Mu:       sync.Mutex{},
			Protocol: protocol,
		}, nil
	default:
		return nil, errors.New(fmt.Sprintf("unknown protocol: %v", protocol))
	}
}
