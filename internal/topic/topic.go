package topic

// Message represents a single record in the queue.
type Message struct {
	Offset uint32
	Key    []byte
	Value  []byte
}

// TopicBuffer defines the contract for appending and reading messages in memory.
// Teammate B will use this to store inbound messages and fetch backlogs.
type TopicBuffer interface {
	// Append adds a message and assigns it the next available offset.
	Append(key, value []byte) uint32
	
	// ReadFrom returns all messages starting exactly at 'offset'.
	ReadFrom(offset uint32) []Message
}

// Buffer is the actual struct you will implement.
type Buffer struct {
	// TODO: Add your slice/ring-buffer and sync.RWMutex here later
}

// Ensure Buffer implements TopicBuffer at compile time.
var _ TopicBuffer = (*Buffer)(nil)

// New creates a new in-memory topic buffer.
func New() *Buffer { 
	return &Buffer{} 
}

// --- Empty Method Stubs (No logic yet) ---

func (b *Buffer) Append(key, value []byte) uint32 { 
	return 0 
}

func (b *Buffer) ReadFrom(offset uint32) []Message { 
	return nil 
}
