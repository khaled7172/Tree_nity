package trie

import (
	"strconv"
	"sync"
	"testing"
)

func TestTriePrefixMatching(t *testing.T) {
	tr := New()

	tr.Add("chat", "Alice")
	tr.Add("chat.gaming", "Bob")
	tr.Add("news", "Charlie")
	tr.Add("", "Admin") // Wildcard subscriber

	// Test 1: Message to "chat.gaming.fps"
	// Should match "chat" (Alice), "chat.gaming" (Bob), and "" (Admin)
	// Should NOT match "news" (Charlie)
	res1 := tr.Match("chat.gaming.fps")
	if len(res1) != 3 {
		t.Errorf("Expected 3 subscribers, got %d", len(res1))
	}

	// Test 2: Message to "chat"
	// Should match "chat" (Alice) and "" (Admin)
	// Bob is "chat.gaming" which is deeper, so he shouldn't get it.
	res2 := tr.Match("chat")
	if len(res2) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", len(res2))
	}
}

func TestTrieRemoveAndPruning(t *testing.T) {
	tr := New()

	tr.Add("cat", "Alice")
	tr.Add("car", "Bob")
	tr.Add("", "Admin")

	// 1. Remove Wildcard
	tr.Remove("", "Admin")
	if len(tr.GlobalSubscribers) != 0 {
		t.Errorf("Failed to remove Admin from GlobalSubscribers")
	}

	// 2. Remove Bob and test ghost branch pruning
	tr.Remove("car", "Bob")
	
	// Since 'car' is empty, the 'r' node should be deleted from 'a'.Children
	// But 'c' and 'a' should still exist because Alice is subscribed to 'cat'
	aNode := tr.Root.Children['c'].Children['a']
	if _, exists := aNode.Children['r']; exists {
		t.Errorf("Failed to prune ghost branch 'r'")
	}
	if _, exists := aNode.Children['t']; !exists {
		t.Errorf("Accidentally pruned 't' branch which belongs to Alice")
	}

	// 3. Remove Alice and prove the whole tree deletes itself back to the root
	tr.Remove("cat", "Alice")
	if len(tr.Root.Children) != 0 {
		t.Errorf("Failed to recursively prune the entire 'cat' branch back to root")
	}
}

func TestTrieConcurrency(t *testing.T) {
	tr := New()
	var wg sync.WaitGroup

	// Spin up 100 simultaneous threads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := "user_" + strconv.Itoa(id)
			
			// Simultaneously Write
			tr.Add("spam.topic", clientID)
			
			// Simultaneously Read
			_ = tr.Match("spam.topic.sub")
			
			// Simultaneously Delete
			tr.Remove("spam.topic", clientID)
		}(i)
	}

	wg.Wait()

	// If the mutex locks work, it won't crash.
	// Furthermore, since every thread added and then removed itself, 
	// the tree should have perfectly pruned itself back to zero!
	if len(tr.Root.Children) != 0 {
		t.Errorf("Expected completely pruned root after concurrent stress test, got %d branches", len(tr.Root.Children))
	}
}
