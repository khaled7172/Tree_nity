package protocol

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEncodeTextMessage(t *testing.T) {
	var buf bytes.Buffer

	key := []byte("sensor.temperature")
	val := []byte("24.5c")

	err := EncodeTextMessage(&buf, key, val)
	if err != nil {
		t.Fatalf("EncodeTextMessage failed unexpectedly: %v", err)
	}

	expected := "sensor.temperature:24.5c\n"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}

	reader := bufio.NewReader(&buff)
	gotKey, gotVal, err := DecodeTextMessage(reader)
	if err != nil {
		t.Fatalf("DecodeTExtMessage failed: %v", err)
	}

	if !bytes.Equal(gotKey, key) {
		t.Errorf("got key %q, want %q", gotKey, key)
	}
	if !bytes.Equal(gotVal, val) {
		t.Errorf("got val %q, want %q", gotVal, val)
	}
}
