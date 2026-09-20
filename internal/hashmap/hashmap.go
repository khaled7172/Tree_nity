package hashmap

import "sync"

type ClientMetadata struct {
	ClientID string
	Topic    string
	Offset   uint32
	Prefix   string
	IPCPath  string
	IsActive bool
}

type ClientMap interface {
	Put(client ClientMetadata) error
	Get(clientID string) (ClientMetadata, bool)
	Remove(clientID string)
	UpdateOffset(clientID string, newOffset uint32) error
}

type Node struct {
	Client ClientMetadata
	Next   *Node
}

type Map struct {
	mu      sync.RWMutex
	buckets []*Node
	size    uint32
}

var _ ClientMap = (*Map)(nil)

func New() *Map {
	return &Map{
		buckets: make([]*Node, 16), // Start with 16 empty buckets
		size:    0,                 // 0 clients currently stored
	}
}

func (m *Map) Put(client ClientMetadata) error {
	return nil
}

func (m *Map) Get(clientID string) (ClientMetadata, bool) {
	return ClientMetadata{}, false
}

func (m *Map) Remove(clientID string) {}

func (m *Map) UpdateOffset(clientID string, newOffset uint32) error {
	return nil
}
