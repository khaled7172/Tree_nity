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

type Map struct {
	mu sync.RWMutex
}

var _ ClientMap = (*Map)(nil)

func New() *Map {
	return &Map{}
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
