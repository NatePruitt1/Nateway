package domain

type Trie struct {
	Root *TrieNode
}

type TrieNode struct {
	Children map[byte]*TrieNode
	Datum    *Endpoint
}

func NewTrie() *Trie {
	return &Trie{
		Root: NewTrieNode(),
	}
}

func NewTrieNode() *TrieNode {
	return &TrieNode{
		Children: make(map[byte]*TrieNode, 26*2),
		Datum:    nil,
	}
}

func (trie *Trie) Add(key []byte, val *Endpoint) {
	if len(key) == 0 {
		return
	}

	if trie.Root == nil {
		trie.Root = NewTrieNode()
	}

	curr := trie.Root

	for i := range len(key) {
		next := curr.Children[key[i]]

		if next == nil {
			next = NewTrieNode()
			curr.Children[key[i]] = next
		}

		curr = next
	}

	curr.Datum = val
}

func (trie *Trie) Find(key []byte) *Endpoint {

	if len(key) == 0 {
		return nil
	}

	curr := trie.Root

	// Set the last valid datum (fallback value) to the root.
	lastValid := trie.Root.Children['/'].Datum

	for i := range len(key) {
		next := curr.Children[key[i]]

		if next == nil {
			return lastValid
		}

		if next.Datum != nil {
			// Lookahead: If the next character is either nothing or a /, then we can use the path stored here.
			if i+1 >= len(key) || key[i+1] == '/' {
				lastValid = next.Datum
			}
		}

		curr = next
	}

	if curr.Datum == nil {
		return lastValid
	} else {
		return curr.Datum
	}
}
