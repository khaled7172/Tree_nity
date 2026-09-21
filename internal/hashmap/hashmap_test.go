package hashmap

import (
	"strconv"
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
