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
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.size >= uint32(len(m.buckets)*3/4) {
		m.resize()
	}

	hash := hashFNV1a(client.ClientID)
	bucketIndex := hash % uint64(len(m.buckets))

	current := m.buckets[bucketIndex]
	for current != nil {
		if current.Client.ClientID == client.ClientID {
			current.Client = client
			return nil
		}
		current = current.Next
	}

	newNode := &Node{
		Client: client,
		Next:   m.buckets[bucketIndex],
	}
	m.buckets[bucketIndex] = newNode
	m.size++

	return nil
}

func (m *Map) Get(clientID string) (ClientMetadata, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hash := hashFNV1a(clientID)
	bucketIndex := hash % uint64(len(m.buckets))

	current := m.buckets[bucketIndex]
	for current != nil {
		if current.Client.ClientID == clientID {
			return current.Client, true
		}
		current = current.Next
	}

	return ClientMetadata{}, false
}

func (m *Map) Remove(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := hashFNV1a(clientID)
	bucketIndex := hash % uint64(len(m.buckets))

	current := m.buckets[bucketIndex]
	var prev *Node

	for current != nil {
		if current.Client.ClientID == clientID {
			if prev == nil {
				m.buckets[bucketIndex] = current.Next
			} else {
				prev.Next = current.Next
			}
			m.size--
			return
		}
		prev = current
		current = current.Next
	}
}

func (m *Map) UpdateOffset(clientID string, newOffset uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := hashFNV1a(clientID)
	bucketIndex := hash % uint64(len(m.buckets))

	current := m.buckets[bucketIndex]
	for current != nil {
		if current.Client.ClientID == clientID {
			current.Client.Offset = newOffset
			return nil
		}
		current = current.Next
	}
	return nil
}

func (m *Map) resize() {
	newCapacity := len(m.buckets) * 2
	newBuckets := make([]*Node, newCapacity)

	for i := 0; i < len(m.buckets); i++ {
		current := m.buckets[i]
		for current != nil {
			next := current.Next

			hash := hashFNV1a(current.Client.ClientID)
			newIndex := hash % uint64(newCapacity)

			current.Next = newBuckets[newIndex]
			newBuckets[newIndex] = current

			current = next
		}
	}

	m.buckets = newBuckets
}
