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
	tr.Add("", "Admin")

	res1 := tr.Match("chat.gaming.fps")
	if len(res1) != 3 {
		t.Errorf("Expected 3 subscribers, got %d", len(res1))
	}

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

	tr.Remove("", "Admin")
	if len(tr.GlobalSubscribers) != 0 {
		t.Errorf("Failed to remove Admin from GlobalSubscribers")
	}

	tr.Remove("car", "Bob")

	aNode := tr.Root.Children['c'].Children['a']
	if _, exists := aNode.Children['r']; exists {
		t.Errorf("Failed to prune ghost branch 'r'")
	}
	if _, exists := aNode.Children['t']; !exists {
		t.Errorf("Accidentally pruned 't' branch which belongs to Alice")
	}

	tr.Remove("cat", "Alice")
	if len(tr.Root.Children) != 0 {
		t.Errorf("Failed to recursively prune the entire 'cat' branch back to root")
	}
}

func TestTrieConcurrency(t *testing.T) {
	tr := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := "user_" + strconv.Itoa(id)

			tr.Add("spam.topic", clientID)
			_ = tr.Match("spam.topic.sub")
			tr.Remove("spam.topic", clientID)
		}(i)
	}

	wg.Wait()

	if len(tr.Root.Children) != 0 {
		t.Errorf("Expected completely pruned root after concurrent stress test, got %d branches", len(tr.Root.Children))
	}
}
