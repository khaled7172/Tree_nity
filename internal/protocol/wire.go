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
	ErrPayloadTooLarge = errors.New("message payload exceeds maximum allowed size (1024 bytes)")
)

// writes a message in text format: key:value\n
func EncodeTextMessage(w io.Writer, key, value []byte) error {
	if len(key)+len(value) > MaxMessagePayload {
		return ErrPayloadTooLarge
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
		return nil, nil, fmt.Errorf("invalid text message format, missing colon separator")
	}

	key := line[:idx]
	value := line[idx+1:]

	if len(key)+len(value) > MaxMessagePayload {
		return nil, nil, ErrPayloadTooLarge
	}

	return key, value, nil
}

func EncodeBinaryProducer(w io.Writer, key, value []byte) error {
	if len(key)+len(value) > MaxMessagePayload {
		return ErrPayloadTooLarge
	}

	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(value)))

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
	var keyLen uint32
	err := binary.Read(r, binary.LittleEndian, &keyLen)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil, io.EOF // clean eof stream ended before any bytes of a new record
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

	var valLen uint32
	if err := binary.Read(r, binary.LittleEndian, &valLen); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, nil, ErrPartialRecord
		}
		return nil, nil, err
	}

	if int(keyLen)+int(valLen) > MaxMessagePayload {
		return nil, nil, ErrPayloadTooLarge
	}

	val := make([]byte, valLen)
	if _, err := io.ReadFull(r, val); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, nil, ErrPartialRecord
		}
		return nil, nil, err
	}

	return key, val, nil
}

func EncodeBinaryConsumer(w io.Writer, offset uint32, key, value []byte) error {
	if len(key)+len(value) > MaxMessagePayload {
		return ErrPayloadTooLarge
	}

	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:4], offset)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(key)))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(value)))

	if _, err := w.Write(buf[0:4]); err != nil {
		return err
	}
	if _, err := w.Write(buf[4:8]); err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	if _, err := w.Write(buf[8:12]); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	return nil
}

// reads consumer message:
func DecodeBinaryConsumer(r io.Reader) (uint32, []byte, []byte, error) {
	var offset uint32
	if err := binary.Read(r, binary.LittleEndian, &offset); err != nil {
		if errors.Is(err, io.EOF) {
			return 0, nil, nil, io.EOF
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, nil, nil, ErrPartialRecord
		}
		return 0, nil, nil, err
	}

	key, val, err := DecodeBinaryProducer(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, nil, nil, ErrPartialRecord
		}
		return 0, nil, nil, err
	}
	return offset, key, val, nil
}
