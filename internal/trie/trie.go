package trie

import "sync"

type Matcher interface {
	Add(prefix string, clientID string)
	Remove(prefix string, clientID string)
	Match(messageKey string) []string
}

type TrieNode struct {
	Children    map[rune]*TrieNode
	Subscribers []string
}

type PrefixTrie struct {
	mu                sync.RWMutex
	Root              *TrieNode
	GlobalSubscribers []string
}

var _ Matcher = (*PrefixTrie)(nil)

func New() *PrefixTrie {
	return &PrefixTrie{
		Root: &TrieNode{
			Children: make(map[rune]*TrieNode),
		},
	}
}

func (t *PrefixTrie) Add(prefix string, clientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if prefix == "" {
		for _, id := range t.GlobalSubscribers {
			if id == clientID {
				return
			}
		}
		t.GlobalSubscribers = append(t.GlobalSubscribers, clientID)
		return
	}

	curr := t.Root
	for _, ch := range prefix {
		next, exists := curr.Children[ch]
		if !exists {
			next = &TrieNode{
				Children: make(map[rune]*TrieNode),
			}
			curr.Children[ch] = next
		}
		curr = next
	}

	for _, id := range curr.Subscribers {
		if id == clientID {
			return
		}
	}
	curr.Subscribers = append(curr.Subscribers, clientID)
}

func (t *PrefixTrie) Remove(prefix string, clientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 1. Handle Global Subscribers
	if prefix == "" {
		for i, id := range t.GlobalSubscribers {
			if id == clientID {
				// Fast O(1) removal by swapping with the last element
				lastIdx := len(t.GlobalSubscribers) - 1
				t.GlobalSubscribers[i] = t.GlobalSubscribers[lastIdx]
				t.GlobalSubscribers = t.GlobalSubscribers[:lastIdx]
				return
			}
		}
		return
	}

	// 2. Traverse and record the path for potential pruning
	path := make([]*TrieNode, 0, len(prefix)+1)
	curr := t.Root
	path = append(path, curr)

	for _, ch := range prefix {
		next, exists := curr.Children[ch]
		if !exists {
			return // Prefix doesn't even exist in the tree
		}
		curr = next
		path = append(path, curr)
	}

	// 3. Remove the clientID from the final node's Subscribers
	targetNode := path[len(path)-1]
	found := false
	for i, id := range targetNode.Subscribers {
		if id == clientID {
			lastIdx := len(targetNode.Subscribers) - 1
			targetNode.Subscribers[i] = targetNode.Subscribers[lastIdx]
			targetNode.Subscribers = targetNode.Subscribers[:lastIdx]
			found = true
			break
		}
	}

	if !found {
		return // Client wasn't subscribed here
	}

	// 4. Prune "Ghost Branches" going backwards
	for i := len(path) - 1; i > 0; i-- {
		child := path[i]
		parent := path[i-1]
		ch := rune(prefix[i-1])

		// If the node is completely empty (no subscribers, no children)
		if len(child.Subscribers) == 0 && len(child.Children) == 0 {
			delete(parent.Children, ch) // Delete the branch from RAM
		} else {
			break // Stop pruning if the node is still being used by others
		}
	}
}

func (t *PrefixTrie) Match(messageKey string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	unique := make(map[string]struct{})
	var result []string

	add := func(id string) {
		if _, exists := unique[id]; !exists {
			unique[id] = struct{}{}
			result = append(result, id)
		}
	}

	for _, id := range t.GlobalSubscribers {
		add(id)
	}

	curr := t.Root
	for _, ch := range messageKey {
		next, exists := curr.Children[ch]
		if !exists {
			break
		}
		curr = next
		for _, id := range curr.Subscribers {
			add(id)
		}
	}

	return result
}
