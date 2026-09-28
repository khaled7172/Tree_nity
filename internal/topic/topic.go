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
	s.mu.RLock()
	td, exists := s.topics[topic]
	s.mu.RUnlock()

	if !exists {
		s.mu.Lock()
		td, exists = s.topics[topic]
		if !exists {
			td = &TopicData{}
			s.topics[topic] = td
		}
		s.mu.Unlock()
	}

	td.mu.Lock()
	defer td.mu.Unlock()

	offset := td.NextOffset
	td.Messages = append(td.Messages, Message{
		Offset:  offset,
		Payload: payload,
	})
	td.NextOffset++

	return offset
}

func (s *Store) ReadFrom(topic string, offset uint32) []Message {
	return nil
}
