package ipc

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestEndpointHelpers(t *testing.T) {
	pid := 1337
	if got := ServerEndpoint(pid); got != "/tmp/operator.server.1337" {
		t.Fatalf("unexpected server endpoint: %s", got)
	}

	if got := ConsumerEndpoint(pid, "client0"); got != "/tmp/operator.server.1337.client0" {
		t.Fatalf("unexpected consumer endpoint: %s", got)
	}

	if got := ClientReplyEndpoint(4242, 1); got != "/tmp/operator.client.4242.1" {
		t.Fatalf("unexpected reply endpoint: %s", got)
	}
}

func TestConsumerChannelPushAndRead(t *testing.T) {
	pid := os.Getpid()
	clientID := "test_subscriber"

	ch, err := NewConsumerChannel(pid, clientID)
	if err != nil {
		t.Fatalf("NewConsumerChannel failed: %v", err)
	}
	defer ch.Close()

	if !IsFifo(ch.Path) {
		t.Fatalf("expected %s to be an active FIFO", ch.Path)
	}

	payload := []byte("alert.critical:fire detected\n")

	if err := ch.Push(payload); err != nil {
		t.Fatalf("ch.Push failed: %v", err)
	}

	r, err := OpenReader(ch.Path)
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer r.Close()

	buf := make([]byte, len(payload))
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if !bytes.Equal(buf[:n], payload) {
		t.Fatalf("expected payload %q, got %q", payload, buf[:n])
	}
}

func TestWriteReplySuccess(t *testing.T) {
	tmpDir := t.TempDir()
	replyPath := filepath.Join(tmpDir, "reply.fifo")

	if err := MakeFifo(replyPath); err != nil {
		t.Fatalf("MakeFifo failed: %v", err)
	}
	defer RemoveFifo(replyPath)

	expectedMsg := []byte("topic created\n")
	readBuf := make([]byte, len(expectedMsg))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		r, err := OpenReader(replyPath)
		if err != nil {
			t.Errorf("OpenReader failed: %v", err)
			return
		}
		defer r.Close()

		_, err = r.Read(readBuf)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond) // Let reader start
		if err := WriteReplyWithTimeout(replyPath, expectedMsg, 2*time.Second); err != nil {
			t.Errorf("WriteReplyWithTimeout failed: %v", err)
		}
	}()

	wg.Wait()

	if !bytes.Equal(readBuf, expectedMsg) {
		t.Fatalf("expected %q, got %q", expectedMsg, readBuf)
	}
}

func TestWriteReplyTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	replyPath := filepath.Join(tmpDir, "reply_timeout.fifo")

	if err := MakeFifo(replyPath); err != nil {
		t.Fatalf("MakeFifo failed: %v", err)
	}
	defer RemoveFifo(replyPath)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WriteReply(ctx, replyPath, []byte("data"))
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}
