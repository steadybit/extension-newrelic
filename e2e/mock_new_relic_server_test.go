package e2e

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

func createMockNewRelicServer() *httptest.Server {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	server := httptest.Server{
		Listener: listener,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().Str("path", r.URL.Path).Str("method", r.Method).Str("query", r.URL.RawQuery).Msg("Mock Server received Request")
			requestBodyBytes, errRead := io.ReadAll(r.Body)
			if errRead != nil {
				panic(errRead)
			}
			requestBody := string(requestBodyBytes)
			if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "actor {accounts {id}}") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(accounts())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "guid name permalink") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(workloads())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "status {value}") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(workloadStatus())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "incidents") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(incidents())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "tags {key values}") && strings.Contains(requestBody, "entity-1") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tagsMatching())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "tags {key values}") && strings.Contains(requestBody, "entity-2") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tagsEmpty())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "alertsMutingRuleCreate") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(mutingRuleCreated())
			} else if strings.HasPrefix(r.URL.Path, "/graphql") && strings.Contains(requestBody, "alertsMutingRuleDelete") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(mutingRuleDeleted())
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		})},
	}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Server")
	return &server
}

func accounts() []byte {
	return []byte(`{
  "data": {
    "actor": {
      "accounts": [
        {
          "id": 12345678
        }
      ]
    }
  }
}`)
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

func incidents() []byte {
	return []byte(`{
  "data": {
    "actor": {
      "account": {
        "aiIssues": {
          "incidents": {
             "incidents": [
                 {
                   "description": [
                     "Policy: 'CPU load'. Condition: 'CPU load'"
                   ],
                   "entityGuids": "entity-1",
                   "entityNames": "ip-10-40-85-195.eu-central-1.compute.internal",
                   "incidentId": "incident-id-1",
                   "priority": "CRITICAL",
                   "title": "[\"CPU % > 20.0 for at least 1 minutes on 'ip-10-40-85-195.eu-central-1.compute.internal'\"]"
                 },
                 {
                   "description": [
                     "Should be ignored - missing tags"
                   ],
                   "entityGuids": "entity-2",
                   "entityNames": "ip-10-40-85-195.eu-central-1.compute.internal",
                   "incidentId": "incident-id-2",
                   "priority": "CRITICAL",
                   "title": "[\"CPU % > 20.0 for at least 1 minutes on 'ip-10-40-85-195.eu-central-1.compute.internal'\"]"
                 }
             ]
          }
        }
      }
    }
  }
}`)
}

func tagsMatching() []byte {
	return []byte(`{
  "data": {
    "actor": {
      "entities": [
        {
          "tags": [
            {
              "key": "my-tag",
              "values": [
                "my-value"
              ]
            }
          ]
        }
      ]
    }
  }
}`)
}

func tagsEmpty() []byte {
	return []byte(`{
  "data": {
    "actor": {
      "entities": [
        {
          "tags": [
          ]
        }
      ]
    }
  }
}`)
}

func workloadStatus() []byte {
	return []byte(`{
    "data": {
        "actor": {
            "account": {
                "workload": {
                    "collection": {
                        "status": {
                            "value": "OPERATIONAL"
                        }
                    }
                }
            }
        }
    }
}`)
}

func mutingRuleCreated() []byte {
	return []byte(`{
    "data": {
        "alertsMutingRuleCreate": {
            "id": "248760"
        }
    }
}`)
}
func mutingRuleDeleted() []byte {
	return []byte(`{
  "data": {
    "alertsMutingRuleDelete": {
      "id": "248760"
    }
  }
}`)
}
