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
			Children:    make(map[rune]*TrieNode),
			Subscribers: []string{},
		},
		GlobalSubscribers: []string{},
	}
}

func (t *PrefixTrie) Add(prefix string, clientID string) {}

func (t *PrefixTrie) Remove(prefix string, clientID string) {}

func (t *PrefixTrie) Match(messageKey string) []string {
	return nil
}
