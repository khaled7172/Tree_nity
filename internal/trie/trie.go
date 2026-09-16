package trie

type Matcher interface {
	Add(prefix string, clientID string)
	Remove(prefix string, clientID string)
	Match(messageKey string) []string
}

type PrefixTrie struct {
}

var _ Matcher = (*PrefixTrie)(nil)

func New() *PrefixTrie {
	return &PrefixTrie{}
}

func (t *PrefixTrie) Add(prefix string, clientID string) {}

func (t *PrefixTrie) Remove(prefix string, clientID string) {}

func (t *PrefixTrie) Match(messageKey string) []string {
	return nil
}
