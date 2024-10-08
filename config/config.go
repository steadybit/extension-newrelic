/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/types"
	"io"
	"net/http"
	"time"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	// The New Relic Base Url, like 'https://api.newrelic.com' or 'https://api.eu.newrelic.com'
	ApiBaseUrl string `json:"apiBaseUrl" split_words:"true" required:"true"`
	// The New Relic API Key
	ApiKey string `json:"apiKey" split_words:"true" required:"true"`
	// The New Relic Insights Base Url, like 'https://insights-collector.newrelic.com' or 'https://insights-collector.eu01.nr-data.net'
	InsightsCollectorApiBaseUrl string `json:"insightsCollectorApiBaseUrl" split_words:"true" required:"true"`
	// The New Relic API Key of type "INGEST - LICENSE"
	InsightsCollectorApiKey string `json:"insightsCollectorApiKey" split_words:"true" required:"true"`
}

var (
	Config Specification
)

func ParseConfiguration() {
	err := envconfig.Process("steadybit_extension", &Config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse configuration from environment.")
	}
}

func ValidateConfiguration() {
	// You may optionally validate the configuration here.
}

const accountsQuery = `{actor {accounts {id}}}`

func (s *Specification) GetAccountIds(_ context.Context) ([]int64, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", accountsQuery)), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get accounts from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.GraphQlResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Errors != nil && len(*result.Errors) > 0 {
			log.Warn().Msgf("API returned errors %+v", result.Errors)
		}

		accounts := make([]int64, 0, len(result.Data.Actor.Accounts))
		for _, account := range result.Data.Actor.Accounts {
			accounts = append(accounts, account.Id)
		}

		return accounts, err
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

const workloadQuery = `{actor {account(id: %d){workload {collections {guid name permalink}}}}}`

func (s *Specification) GetWorkloads(_ context.Context, accountId int64) ([]types.Workload, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(workloadQuery, accountId))), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get workloads from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.GraphQlResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Errors != nil && len(*result.Errors) > 0 {
			log.Warn().Msgf("API returned errors %+v", result.Errors)
		}
		return result.Data.Actor.Account.Workload.Collections, err
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

const workloadStatusQuery = `{actor {account(id: %d){ workload { collection(guid: \"%s\") {status {value}}}}}}`

func (s *Specification) GetWorkloadStatus(_ context.Context, workloadGuid string, accountId int64) (*string, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(workloadStatusQuery, accountId, workloadGuid))), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Str("workloadGuid", workloadGuid).Msgf("Failed to get workload status from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Str("workloadGuid", workloadGuid).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.GraphQlResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Errors != nil && len(*result.Errors) > 0 {
			log.Warn().Msgf("API returned errors %+v", result.Errors)
		}
		if result.Data.Actor.Account.Workload.Collection != nil && result.Data.Actor.Account.Workload.Collection.Status != nil {
			return &result.Data.Actor.Account.Workload.Collection.Status.Value, err
		}
		//Workaround - New Relic has regular timeouts
		//{"data":{"actor":{"account":{"workload":{"collection":null}}}},"errors":[{"extensions":{"errorClass":"TIMEOUT"},"locations":[{"column":42,"line":1}],"message":"Resolution of this field timed out","path":["actor","account","workload","collection"]}]}
		log.Warn().Err(err).Str("body", string(responseBody)).Msgf("Unexpected response body, return status UNKNOWN")
		return extutil.Ptr("UNKNOWN"), nil
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

const mutingRuleCreate = `mutation{alertsMutingRuleCreate(accountId: %d rule: {condition: {conditions: {attribute: \"accountId\", operator: EQUALS, values: \"%d\"}, operator: AND}, name: \"%s\", schedule: {endTime: \"%s\", timeZone: \"UTC\"}, description: \"%s\", enabled: true}  ) {id}}`

func (s *Specification) CreateMutingRule(_ context.Context, accountId int64, name string, description string, end time.Time) (*string, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)
	endString := end.UTC().Format("2006-01-02T15:04:05")
	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(mutingRuleCreate, accountId, accountId, name, endString, description))), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create muting rule in New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.GraphQlResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Data != nil && result.Data.AlertsMutingRuleCreate != nil {
			return &result.Data.AlertsMutingRuleCreate.Id, err
		}
		log.Error().Err(err).Msgf("Unexpected response body %+v", string(responseBody))
		return nil, errors.New("unexpected response body")
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

const mutingRuleDelete = `mutation {alertsMutingRuleDelete(id: %s, accountId: %d){id}}`

func (s *Specification) DeleteMutingRule(_ context.Context, accountId int64, mutingRuleId string) error {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)
	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(mutingRuleDelete, mutingRuleId, accountId))), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to delete muting rule in New Relic. Full response %+v", string(responseBody))
		return err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return errors.New("unexpected response code")
	}

	return nil
}

const entityTagsQuery = `{actor {entities(guids: \"%s\"){tags {key values}}}}`

func (s *Specification) GetEntityTags(_ context.Context, guid string) (map[string][]string, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(entityTagsQuery, guid))), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get entity tags from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.GraphQlResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Errors != nil && len(*result.Errors) > 0 {
			log.Warn().Msgf("API returned errors %+v", result.Errors)
		}

		if len(result.Data.Actor.Entities) == 1 {
			tags := make(map[string][]string)
			for _, tag := range result.Data.Actor.Entities[0].Tags {
				tags[tag.Key] = tag.Values
			}
			return tags, err
		}
		log.Error().Err(err).Msgf("Unexpected response body %+v", string(responseBody))
		return nil, errors.New("unexpected response body")
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

const incidentsQuery = `{actor {account(id: %d){aiIssues {incidents(filter: {priority: [%s], states: CREATED}) {incidents {incidentId entityGuids entityNames title description priority}}}}}}`

func (s *Specification) GetIncidents(_ context.Context, incidentPriorityFilter []string, accountId int64) ([]types.Incident, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	priorityFilter := ""
	for i, priority := range incidentPriorityFilter {
		if i > 0 {
			priorityFilter += ","
		}
		priorityFilter += fmt.Sprintf("\\\"%s\\\"", priority)
	}
	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(incidentsQuery, accountId, priorityFilter))), s.ApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get incidents from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.GraphQlResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Errors != nil && len(*result.Errors) > 0 {
			log.Warn().Msgf("API returned errors %+v", result.Errors)
		}
		if result.Data != nil && result.Data.Actor.Account != nil && result.Data.Actor.Account.AiIssues != nil && result.Data.Actor.Account.AiIssues.Incidents != nil {
			return result.Data.Actor.Account.AiIssues.Incidents.Incidents, err
		}
		return []types.Incident{}, nil
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

func (s *Specification) PostEvent(_ context.Context, event *types.EventIngest, accountId int64) error {
	url := fmt.Sprintf("%s/v1/accounts/%d/events", s.InsightsCollectorApiBaseUrl, accountId)

	objects := []types.EventIngest{*event}

	b, err := json.Marshal(objects)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal request")
		return err
	}

	responseBody, response, err := s.do(url, "POST", b, s.InsightsCollectorApiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to post event to New Relic. Full response %+v", string(responseBody))
		return err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return errors.New("unexpected response code")
	}

	return nil
}

func (s *Specification) do(url string, method string, body []byte, apiKey string) ([]byte, *http.Response, error) {
	log.Debug().Str("url", url).Str("method", method).Msg("Requesting New Relic API")
	if body != nil {
		log.Debug().Int("len", len(body)).Str("body", string(body)).Msg("Request body")
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	request, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create request")
		return nil, nil, err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("API-Key", apiKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to execute request")
		return nil, response, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to close response body")
		}
	}(response.Body)

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read body")
		return nil, response, err
	}

	return responseBody, response, err
}
