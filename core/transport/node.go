package transport

import (
	"sync"
)

type Node struct {
	Trans    Transport
	Mu       sync.Mutex
	Protocol Proto
}
