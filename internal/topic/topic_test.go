package topic

import (
	"bytes"
	"strconv"
	"sync"
	"testing"
)

func TestAppendAndRead(t *testing.T) {
	store := New()

	store.Append("chat", []byte("msg0"))
	store.Append("chat", []byte("msg1"))
	store.Append("chat", []byte("msg2"))

	resAll := store.ReadFrom("chat", 0)
	if len(resAll) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(resAll))
	}

	resPartial := store.ReadFrom("chat", 1)
	if len(resPartial) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(resPartial))
	}
	if !bytes.Equal(resPartial[0].Payload, []byte("msg1")) {
		t.Errorf("Expected msg1 payload")
	}
}

func TestReadFromEdgeCases(t *testing.T) {
	store := New()
	store.Append("chat", []byte("hello"))

	res1 := store.ReadFrom("ghost", 0)
	if res1 != nil {
		t.Errorf("Expected nil for non-existent topic")
	}

	res2 := store.ReadFrom("chat", 5)
	if res2 != nil {
		t.Errorf("Expected nil for future offset")
	}

	res3 := store.ReadFrom("chat", 1)
	if res3 != nil {
		t.Errorf("Expected nil when fully caught up")
	}
}

func TestTopicConcurrency(t *testing.T) {
	store := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)

		go func(id int) {
			defer wg.Done()
			payload := []byte("payload_" + strconv.Itoa(id))
			store.Append("spam", payload)
		}(i)

		go func() {
			defer wg.Done()
			_ = store.ReadFrom("spam", 0)
		}()
	}

	wg.Wait()

	finalRes := store.ReadFrom("spam", 0)
	if len(finalRes) != 100 {
		t.Errorf("Expected exactly 100 messages, got %d", len(finalRes))
	}
}
