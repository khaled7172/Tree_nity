package protocol

import (
	"errors"
	"io"
)

var (
	// this is returned when a stream terminates midway through a record.
	ErrPartialRecord = errors.New("partial record at EOF")

	// this is returned when key + value exceeds 1024 bytes.
	ErrPayLoadTooLarge = errors.New("message payload exceeds maximum allowed size (1024 bytes)")
)

// writes a message in text format: key:value\n
func EncodeTextMessage(w io.Writer, key, value []byte) error {
	if len(key)+len(value) > MaxMessagePayload {
		return ErrPayLoadTooLarge
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	if _, err := w.Write([]byte(":")); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}
