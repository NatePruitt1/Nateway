package domain

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
)

type LongestMatchRouterCFG struct {
	Routes map[string]Endpoint `json:"routes"`
}

type LongestMatchRouterLoader struct {
	longestMatchRouter atomic.Pointer[LongestMatchRouter]
	fileLocation       string
}

func NewLongestMatchRouterLoader(fileLocation string) (*LongestMatchRouterLoader, error) {
	loader := LongestMatchRouterLoader{fileLocation: fileLocation}
	err := loader.Load()

	if err != nil {
		return nil, err
	}
	return &loader, nil
}

func (lmrl *LongestMatchRouterLoader) Pin() *LongestMatchRouter {
	return lmrl.longestMatchRouter.Load()
}

func (lmrl *LongestMatchRouterLoader) Load() error {
	content, err := os.ReadFile(lmrl.fileLocation)
	if err != nil {
		return err
	}

	var newCFG LongestMatchRouterCFG
	err = json.Unmarshal(content, &newCFG)
	if err != nil {
		return err
	}

	newRouter := NewLongestMatchRouter(newCFG)
	lmrl.longestMatchRouter.Store(newRouter)

	log.Printf("Longest Match Router Loaded: %+v\n", newRouter)
	for k, e := range newRouter.addressMap {
		log.Printf("%s: %+v\n", k, e)
	}

	return nil
}

func (lmrl *LongestMatchRouterLoader) Watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				log.Println("Event:", event)
				if event.Has(fsnotify.Write) {
					log.Println("File changed, reloading...")
					err := lmrl.Load()
					if err != nil {
						log.Println("Error reloading:", err)
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	err = watcher.Add(lmrl.fileLocation)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

type LongestMatchRouter struct {
	addressMap map[string]Endpoint
}

func NewLongestMatchRouter(cfg LongestMatchRouterCFG) *LongestMatchRouter {
	return &LongestMatchRouter{addressMap: cfg.Routes}
}

func (lmr *LongestMatchRouter) GetEndpoint(r *http.Request) (*Endpoint, error) {
	// Search for the longest key match
	matchLen := -1
	var longestMatch *Endpoint = nil
	for key, endpoint := range lmr.addressMap {
		currMatch := getMatchLength(key, r.URL.Path)
		if currMatch > matchLen {
			matchLen = currMatch
			longestMatch = &endpoint
		}
	}

	if longestMatch == nil {
		return nil, NoRouteFoundErr
	} else {
		return longestMatch, nil
	}
}

func getMatchLength(s, t string) int {
	for i := 0; i < min(len(s), len(t)); i++ {
		if s[i] != t[i] {
			return i
		}
	}
	return min(len(s), len(t))
}
