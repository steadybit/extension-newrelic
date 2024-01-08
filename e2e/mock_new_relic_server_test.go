package e2e

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

var Requests []string

func createMockNewRelicServer() *httptest.Server {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	server := httptest.Server{
		Listener: listener,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().Str("path", r.URL.Path).Str("method", r.Method).Str("query", r.URL.RawQuery).Msg("Mock Server received Request")
			Requests = append(Requests, fmt.Sprintf("%s-%s", r.Method, r.URL.Path))
			if strings.HasPrefix(r.URL.Path, "/graphql") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(workloads())
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		})},
	}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Server")
	return &server
}
func workloads() []byte {
	return []byte(`{
    "data": {
        "actor": {
            "account": {
                "workload": {
                    "collections": [
                        {
                            "guid": "Mjg0NzgwNnxOUjF8V09SS0xPQUR8Mjg5MzM",
                            "name": "sandbox-demo"
                        }
                    ]
                }
            }
        }
    }
}`)
}
