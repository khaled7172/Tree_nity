package trie

// Matcher defines the contract for our prefix filtering engine.
// Teammate B's topic dispatcher will call this to find out who gets a message.
type Matcher interface {
	// Add registers a consumer's ID under a specific prefix.
	Add(prefix string, clientID string)
	
	// Remove deletes a consumer from the trie.
	Remove(prefix string, clientID string)
	
	// Match returns a list of all clientIDs whose registered prefix matches the messageKey.
	// E.g., messageKey "user.login" matches prefixes "user", "user.", "user.login", and "".
	Match(messageKey string) []string
}

// PrefixTrie is the actual struct you will implement.
type PrefixTrie struct {
	// TODO: Add your Radix/Trie nodes and wildcard slice here later
}

// Ensure PrefixTrie implements Matcher at compile time.
var _ Matcher = (*PrefixTrie)(nil)

// New creates and initializes a new Prefix Trie.
func New() *PrefixTrie { 
	return &PrefixTrie{} 
}

// --- Empty Method Stubs (No logic yet) ---

func (t *PrefixTrie) Add(prefix string, clientID string) {}

func (t *PrefixTrie) Remove(prefix string, clientID string) {}

func (t *PrefixTrie) Match(messageKey string) []string { 
	return nil 
}
