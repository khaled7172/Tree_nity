package hashmap

import (
	"strconv"
	"sync"
	"testing"
)

func TestHashmapPutAndGet(t *testing.T) {
	m := New()

	alice := ClientMetadata{ClientID: "Alice", Topic: "gaming"}
	m.Put(alice)

	// Test successful retrieval
	client, found := m.Get("Alice")
	if !found {
		t.Errorf("Expected to find Alice, but got not found")
	}
	if client.Topic != "gaming" {
		t.Errorf("Expected Topic to be gaming, got %s", client.Topic)
	}

	// Test missing client
	_, found = m.Get("Bob")
	if found {
		t.Errorf("Expected not to find Bob, but found him")
	}
}

func TestHashmapRemove(t *testing.T) {
	m := New()
	m.Put(ClientMetadata{ClientID: "Alice"})
	m.Put(ClientMetadata{ClientID: "Bob"})

	m.Remove("Alice")

	_, found := m.Get("Alice")
	if found {
		t.Errorf("Expected Alice to be removed, but found her")
	}

	_, found = m.Get("Bob")
	if !found {
		t.Errorf("Expected Bob to still be in the map, but he was removed")
	}
}

func TestHashmapDynamicResizing(t *testing.T) {
	m := New()
	initialBuckets := len(m.buckets)

	// Spam 1,000 clients into the map
	for i := 0; i < 1000; i++ {
		clientID := "user_" + strconv.Itoa(i)
		m.Put(ClientMetadata{ClientID: clientID, Topic: "general"})
	}

	// The map should have resized multiple times to handle 1000 users
	finalBuckets := len(m.buckets)
	if finalBuckets <= initialBuckets {
		t.Errorf("Expected buckets to dynamically resize, started with %d, ended with %d", initialBuckets, finalBuckets)
	}

	// Verify we can still successfully retrieve someone
	client, found := m.Get("user_500")
	if !found {
		t.Errorf("Lost user_500 during the resize process")
	}
	if client.Topic != "general" {
		t.Errorf("Data corrupted during resize, expected topic 'general', got '%s'", client.Topic)
	}
}

func TestHashmapConcurrency(t *testing.T) {
	m := New()
	var wg sync.WaitGroup

	// Spin up 100 simultaneous goroutines (threads)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			clientID := "user_" + strconv.Itoa(id)
			
			// Simultaneously Write
			m.Put(ClientMetadata{ClientID: clientID, Topic: "spam"})
			
			// Simultaneously Read
			_, _ = m.Get(clientID)
			
			// Simultaneously Update
			_ = m.UpdateOffset(clientID, 10)
		}(i)
	}

	// Wait for all 100 goroutines to finish crashing into the map
	wg.Wait()

	// If the test reaches this line without panicking, the Mutex successfully protected the memory!
	if m.size != 100 {
		t.Errorf("Expected map size to be 100, got %d", m.size)
	}
}
