package transport

import "strings"

type Proto = byte
type Addr = string

const (
	Unknown Proto = iota
	TCP
	Tor
)

type NodeAddr struct {
	Protocol Proto
	Address  Addr
}

type Handler = func(from NodeAddr, data []byte)

// Transport 只负责选择协议、收发字节
type Transport interface {
	Send(addr NodeAddr, data []byte) error
	OnRecv(handler Handler)
	LocalAddr() NodeAddr
	Close() error
	Protocol() Proto
}

func ProtoToString(p Proto) string {
	switch p {
	case TCP:
		return "TCP"
	case Tor:
		return "Tor"
	default:
		return ""
	}
}

func WhichProtocol(s string) Proto {
	if strings.HasPrefix(s, "tcp://") {
		return TCP
	} else if strings.HasSuffix(s, ".onion") {
		return Tor
	}

	return Unknown
}
