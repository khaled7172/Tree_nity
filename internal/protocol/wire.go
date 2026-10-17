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
	ErrPartialRecord   = errors.New("partial record at EOF")
	ErrPayloadTooLarge = errors.New("message payload exceeds maximum allowed size (1024 bytes)")
)

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

func DecodeTextMessage(r *bufio.Reader) ([]byte, []byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil, nil, io.EOF
			}
		} else {
			return nil, nil, err
		}
	}

	line = bytes.TrimRight(line, "\r\n")
	if len(line) == 0 {
		if errors.Is(err, io.EOF) {
			return nil, nil, io.EOF
		}
		return DecodeTextMessage(r)
	}

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
			return nil, nil, io.EOF
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
