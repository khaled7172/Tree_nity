package topic

import "sync"

type Message struct {
	Offset  uint32
	Payload []byte
}

type TopicData struct {
	mu         sync.RWMutex
	Messages   []Message
	NextOffset uint32
}

type TopicStore interface {
	Append(topic string, payload []byte) uint32
	ReadFrom(topic string, offset uint32) []Message
}

type Store struct {
	mu     sync.RWMutex
	topics map[string]*TopicData
}

var _ TopicStore = (*Store)(nil)

func New() *Store {
	return &Store{
		topics: make(map[string]*TopicData),
	}
}

func (s *Store) Append(topic string, payload []byte) uint32 {
	return 0
}

func (s *Store) ReadFrom(topic string, offset uint32) []Message {
	return nil
}
