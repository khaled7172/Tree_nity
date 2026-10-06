package engine

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"Tree_nity/internal/ipc"
	"Tree_nity/internal/topic"
	"Tree_nity/internal/trie"
)

func TestTopicWorkerMessageDispatch(t *testing.T) {
	store := topic.New()
	matcher := trie.New()
	worker := NewTopicWorker("orders", store, matcher)
	worker.Start()
	defer worker.Stop()

	serverPID := os.Getpid()

	ch1, err := ipc.NewConsumerChannel(serverPID, "client_filter")
	if err != nil {
		t.Fatalf("failed to create ch1: %v", err)
	}
	defer ch1.Close()

	ch2, err := ipc.NewConsumerChannel(serverPID, "client_wildcard")
	if err != nil {
		t.Fatalf("failed to create ch2: %v", err)
	}
	defer ch2.Close()

	if err := worker.Subscribe("client_filter", "order.created", 0, ch1); err != nil {
		t.Fatalf("subscribe ch1 failed: %v", err)
	}
	if err := worker.Subscribe("client_wildcard", "", 0, ch2); err != nil {
		t.Fatalf("subscribe ch2 failed: %v", err)
	}

	r1, err := ipc.OpenReader(ch1.Path)
	if err != nil {
		t.Fatalf("open reader 1 failed: %v", err)
	}
	defer r1.Close()

	r2, err := ipc.OpenReader(ch2.Path)
	if err != nil {
		t.Fatalf("open reader 2 failed: %v", err)
	}
	defer r2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msgA := InboundMessage{
		Key:     "order.created",
		Payload: []byte("order.created:1001\n"),
	}
	if err := worker.Publish(ctx, msgA); err != nil {
		t.Fatalf("publish msgA failed: %v", err)
	}

	msgB := InboundMessage{
		Key:     "order.canceled",
		Payload: []byte("order.canceled:1002\n"),
	}
	if err := worker.Publish(ctx, msgB); err != nil {
		t.Fatalf("publish msgB failed: %v", err)
	}

	buf1 := make([]byte, len(msgA.Payload))
	n1, err := r1.Read(buf1)
	if err != nil {
		t.Fatalf("r1 read failed: %v", err)
	}
	if !bytes.Equal(buf1[:n1], msgA.Payload) {
		t.Fatalf("ch1 expected %q, got %q", msgA.Payload, buf1[:n1])
	}

	expectedCh2 := append(msgA.Payload, msgB.Payload...)
	buf2 := make([]byte, len(expectedCh2))
	n2, err := r2.Read(buf2)
	if err != nil {
		t.Fatalf("r2 read failed: %v", err)
	}
	if !bytes.Equal(buf2[:n2], expectedCh2) {
		t.Fatalf("ch2 expected %q, got %q", expectedCh2, buf2[:n2])
	}
}

func TestTopicWorkerHistoricalReplay(t *testing.T) {
	store := topic.New()
	matcher := trie.New()
	worker := NewTopicWorker("events", store, matcher)
	worker.Start()
	defer worker.Stop()

	store.Append("events", []byte("msg0\n"))
	store.Append("events", []byte("msg1\n"))
	store.Append("events", []byte("msg2\n"))

	serverPID := os.Getpid()
	ch, err := ipc.NewConsumerChannel(serverPID, "late_subscriber")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	defer ch.Close()

	if err := worker.Subscribe("late_subscriber", "", 1, ch); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	r, err := ipc.OpenReader(ch.Path)
	if err != nil {
		t.Fatalf("open reader failed: %v", err)
	}
	defer r.Close()

	expected := []byte("msg1\nmsg2\n")
	buf := make([]byte, len(expected))
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read replay failed: %v", err)
	}
	if !bytes.Equal(buf[:n], expected) {
		t.Fatalf("expected replay %q, got %q", expected, buf[:n])
	}
}
