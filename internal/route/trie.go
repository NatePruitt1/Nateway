package route

import "errors"

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

// Strip last /. If this is a root call, leave the special first slash.
func formatPath(path []byte) []byte {
	if len(path) > 1 && path[len(path)-1] == '/' {
		return path[:len(path)-1]
	}

	if len(path) == 0 {
		return []byte("/")
	}

	return path
}

func (trie *Trie) Add(key []byte, val *Endpoint) error {
	pattern := getPathValidationPattern()

	if len(key) == 0 {
		return errors.New("no empty paths.")
	}

	if !pattern.Match(key) {
		return errors.New("Paths must match regex. No empty paths, trailing slashes, or non-letter/number characters.")
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
	return nil
}

func (trie *Trie) Find(path []byte) *Endpoint {
	key := formatPath(path)

	if len(key) == 0 {
		return trie.Find([]byte("/"))
	}

	curr := trie.Root

	for i := range len(key) {
		next := curr.Children[key[i]]

		if next == nil {
			return nil
		}

		curr = next
	}

	return curr.Datum
}
