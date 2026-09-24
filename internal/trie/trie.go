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

func (t *PrefixTrie) Remove(prefix string, clientID string) {}

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
