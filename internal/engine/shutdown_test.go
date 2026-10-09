package engine

import (
	"bytes"
	"os"
	"testing"
	"time"

	"Tree_nity/internal/hashmap"
	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
	"Tree_nity/internal/topic"
)

func TestServerGracefulShutdownSentinel(t *testing.T) {
	store := topic.New()
	clientMap := hashmap.New()
	srv := NewServer(os.Getpid()+3000, store, clientMap)

	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start failed: %v", err)
	}

	_ = sendRequest(t, srv.Endpoint, protocol.Request{
		Type:  protocol.CmdCreate,
		Topic: "metrics",
	})

	_ = sendRequest(t, srv.Endpoint, protocol.Request{
		Type:     protocol.CmdSubscribe,
		Topic:    "metrics",
		ClientID: "consumer_shutdown_test",
	})

	consumerFIFO := ipc.ConsumerEndpoint(srv.PID, "consumer_shutdown_test")
	r, err := ipc.OpenReader(consumerFIFO)
	if err != nil {
		t.Fatalf("open consumer reader failed: %v", err)
	}
	defer r.Close()

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- srv.Shutdown(2 * time.Second)
	}()

	expectedSentinel := []byte(ShutdownSentinel)
	buf := make([]byte, len(expectedSentinel))
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read sentinel failed: %v", err)
	}

	if !bytes.Equal(buf[:n], expectedSentinel) {
		t.Fatalf("expected sentinel %q, got %q", expectedSentinel, buf[:n])
	}

	offset := uint32(10)
	_ = sendRequest(t, srv.Endpoint, protocol.Request{
		Type:     protocol.CmdDrainAck,
		ClientID: "consumer_shutdown_test",
		Offset:   &offset,
	})

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Shutdown timed out")
	}

	if ipc.IsFifo(srv.Endpoint) {
		t.Fatalf("expected server FIFO to be deleted")
	}
	if ipc.IsFifo(consumerFIFO) {
		t.Fatalf("expected consumer FIFO to be deleted")
	}
}
