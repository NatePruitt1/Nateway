package route

import (
	"net/http"
)

type TrieRouter struct {
	PrefixTree *Trie
}

func NewPrefixTreeRouter(epMap map[string]*Endpoint) (*TrieRouter, error) {
	trie := NewTrie()
	for k, v := range epMap {

		if err := trie.Add([]byte(k), v); err != nil {
			return nil, err
		}
	}

	return &TrieRouter{
		PrefixTree: trie,
	}, nil
}

func (tr *TrieRouter) GetEndpoint(r *http.Request) (*Endpoint, error) {
	ep := tr.PrefixTree.Find([]byte(r.URL.Path))

	if ep == nil {
		return nil, ErrNoRoute
	} else {
		return ep, nil
	}
}
