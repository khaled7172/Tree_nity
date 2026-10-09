package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

// this read a single text (key:value\n)
// returns (key, value, nil) on success
// return (nil, nil, io.EOF) when the stream ends cleanly with no data left

func DecodeTextMessage(r *bufio.Reader) ([]byte, []byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil, nil, io.EOF // clean EOF
			}
			// If we got bytes before EOF without trailing \n, still process that line
		} else {
			return nil, nil, err
		}
	}

	// Remove trailing \r or \n
	line = bytes.TrimRight(line, "\r\n")
	if len(line) == 0 {
		if errors.Is(err, io.EOF) {
			return nil, nil, io.EOF
		}
		// skip empty line and read next
		return DecodeTextMessage(r)
	}

	// Find the FIRST colon ':'
	idx := bytes.IndexByte(line, ':')
	if idx == -1 {
		return nil, nil, fmt.Errorf("invalid text messahe format, missing colon seperator")
	}

	key := line[:idx]
	value := line[idx+1:]

	if len(key)+len(value) > MaxMessagePayload {
		return nil, nil, ErrPayLoadTooLarge
	}

	return key, value, nil
}

func EncodeBinaryProducer(w io.Writer, key, value []byte) error {
	if len(key)+len(value) > MaxMessagePayload {
		return ErrPayLoadTooLarge
	}

	header := make([]byte, 8)
	binary.LittleEndian.PutUnit32(header[0:4], unit32(len(key)))
	binary.LittleEndian.PutUint32(header[4:8], unit32(len(value)))

	if _, err := w.Write(header[0:4]); err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	if _, err := w.Write(header[4:8]); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	return nil
}

func DecodeBinaryProducer(r io.Reader) ([]byte, []byte, error) {
	var KeyLen uint32
	err := binary.Read(r, binary.LittleEndian, &KeyLen)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil, io.EOF // clean eod stream ended before any bytes of a new record
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, nil, ErrPartialRecord
		}
		return nil, nil, err
	}

	key := make([]byte, keyLen)
	if _, err := io.ReadFull(r, key); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, nil, ErrPartialRecord
		}
		return nil, nil, err
	}
}
