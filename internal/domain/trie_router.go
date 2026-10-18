package domain

import (
	"net/http"
)

type TrieRouter struct {
	PrefixTree *Trie
}

func NewPrefixTreeRouter(cfg *GatewayConfig) *TrieRouter {
	trie := NewTrie()
	for loc := range cfg.Server.Locations {
		epcfg := cfg.Server.Locations[loc]
		endp := Endpoint{
			Address:      epcfg.Target,
			AuthKey:      epcfg.AuthKey,
			AuthRequired: epcfg.Auth,
		}

		trie.Add([]byte(epcfg.Path), &endp)
	}

	return &TrieRouter{
		PrefixTree: trie,
	}
}

func (tr *TrieRouter) GetEndpoint(r *http.Request) (*Endpoint, error) {
	ep := tr.PrefixTree.Find([]byte(r.URL.Path))

	if ep == nil {
		return nil, NoRouteFoundErr
	} else {
		return ep, nil
	}
}
