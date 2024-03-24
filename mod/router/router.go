package router

import (
	"log"
	"net/http"
	"regexp"
	"strings"
)

type PathValues map[string]string
type endpointHandler = func(w http.ResponseWriter, r *http.Request, values PathValues)
type endpointEntry struct {
	endpoint *regexp.Regexp
	keys     []string
	handler  endpointHandler
}

var endpointEntries []endpointEntry

func RegistorEndpoint(path string, handler endpointHandler) {
	tokens := strings.Split(path, "/")
	keys := []string{}

	for i, t := range tokens {
		if t[0] != ':' {
			continue
		}

		key := t[1:]
		keys = append(keys, key)
		tokens[i] = "(?P<" + key + ">[^/]+)"
	}

	endpointExp := strings.Join(tokens, "/") + "$"
	log.Println("Registored endpoint: " + endpointExp)

	re, err := regexp.Compile(endpointExp)
	if err != nil {
		log.Println("Failed regexp compile: " + path)
		return
	}

	endpointEntries = append(endpointEntries, endpointEntry{
		endpoint: re,
		keys:     keys,
		handler:  handler,
	})
}

var Router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	for _, e := range endpointEntries {
		m := e.endpoint.FindAllStringSubmatch(r.Method+" "+r.URL.Path, -1)
		if len(m) == 0 {
			continue
		}

		log.Println(m, e.keys)

		values := PathValues{}
		for _, key := range e.keys {
			values[key] = m[0][e.endpoint.SubexpIndex(key)]
		}
		e.handler(w, r, values)

		return
	}

	http.NotFound(w, r)
})
