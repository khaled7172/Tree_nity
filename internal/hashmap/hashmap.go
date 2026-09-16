package hashmap

import "sync"

// ClientMetadata holds all information about an active consumer.
// It is stored in your custom hashmap.
type ClientMetadata struct {
	ClientID string // Matches ^[a-zA-Z0-9_.-]{1,32}$
	Topic    string // Matches ^[a-zA-Z0-9_.-]{1,32}$
	Offset   uint32 // Next topic offset the client wants to consume (N + 1)
	Prefix   string // Optional key prefix filter (can be empty "")
	IPCPath  string // Path to the dedicated consumer FIFO
	IsActive bool   // Whether the consumer is actively connected
}

// ClientMap defines the contract for our custom hashmap implementation.
// Teammate B will use these methods to manage client connections.
type ClientMap interface {
	// Put adds or updates a client in the hashmap.
	Put(client ClientMetadata) error
	
	// Get retrieves a client by their ClientID. Returns false if not found.
	Get(clientID string) (ClientMetadata, bool)
	
	// Remove deletes a client from the hashmap.
	Remove(clientID string)
	
	// UpdateOffset updates only the offset for a specific client (performance optimization).
	UpdateOffset(clientID string, newOffset uint32) error
}

// Map is the actual struct you will implement (Separate Chaining).
type Map struct {
	// TODO: Add your buckets array and linked-list nodes here later
	mu sync.RWMutex
}

// Ensure Map implements ClientMap at compile time.
var _ ClientMap = (*Map)(nil)

// New creates and initializes a new custom hashmap.
func New() *Map {
	return &Map{}
}

// --- Empty Method Stubs (No logic yet) ---

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
