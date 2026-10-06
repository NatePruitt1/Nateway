package apigateway

import "net/url"

type RouteEndpoint struct {
	Target url.URL
	Path   string
}

type Routes struct {
	Routes   []string
	RouteMap map[string]RouteEndpoint
}

func (r *Routes) GetLongestPrefixRoute(route string) *RouteEndpoint {
	// Go through each route and find the longest non 0 prefix
	longestInd := 0
	longestPrefix := countPathPrefix(r.Routes[0], route)
	for i := 1; i < len(r.Routes); i++ {
		prefix := countPathPrefix(r.Routes[i], route)
		if longestPrefix < prefix {
			longestPrefix = prefix
			longestInd = i
		}
	}

	if longestPrefix == 0 {
		return nil
	} else {
		val, ok := r.RouteMap[r.Routes[longestInd]]
		if ok {
			return &val
		} else {
			return nil
		}
	}
}

func countPathPrefix(s1, s2 string) int {
	minLen := min(len(s1), len(s2))
	prefix := 0
	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			prefix++
		} else {
			break
		}
	}

	return prefix
}
