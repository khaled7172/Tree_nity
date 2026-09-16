package topic

type Message struct {
	Offset uint32
	Key    []byte
	Value  []byte
}

type TopicBuffer interface {
	Append(key, value []byte) uint32
	ReadFrom(offset uint32) []Message
}

type Buffer struct {
}

var _ TopicBuffer = (*Buffer)(nil)

func New() *Buffer {
	return &Buffer{}
}

func (b *Buffer) Append(key, value []byte) uint32 {
	return 0
}

func (b *Buffer) ReadFrom(offset uint32) []Message {
	return nil
}
