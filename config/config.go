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
	"github.com/steadybit/extension-newrelic/types"
	"io"
	"net/http"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	// The New Relic Base Url, like 'https://api.newrelic.com'
	ApiBaseUrl string `json:"apiBaseUrl" split_words:"true" required:"true"`
	// The New Relic API Key
	ApiKey string `json:"apiKey" split_words:"true" required:"true"`
	// Your New Relic Account Id
	AccountId string `json:"accountId" split_words:"true" required:"true"`
	// The New Relic Insights Base Url, like 'https://insights-api.newrelic.com'
	InsightsApiBaseUrl string `json:"insightsApiBaseUrl" split_words:"true" required:"true"`
	// The New Relic Insights Insert Key
	InsightsInsertKey string `json:"insightsInsertKey" split_words:"true" required:"true"`
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

const workloadQuery = `{actor {account(id: %s){workload {collections {guid name permalink}}}}}`
const workloadStatusQuery = `{actor {account(id: %s){ workload { collection(guid: \"%s\") {status {value}}}}}}`

func (s *Specification) GetWorkloads(_ context.Context) ([]types.Workload, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(workloadQuery, s.AccountId))))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get workloads from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.WorkloadSearchResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		return result.Data.Actor.Account.Workload.Collections, err
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

func (s *Specification) GetWorkloadStatus(ctx context.Context, workloadGuid string) (*string, error) {
	url := fmt.Sprintf("%s/graphql", s.ApiBaseUrl)

	responseBody, response, err := s.do(url, "POST", []byte(fmt.Sprintf("{\"query\": \"%s\"}", fmt.Sprintf(workloadStatusQuery, s.AccountId, workloadGuid))))
	if err != nil {
		log.Error().Err(err).Str("workloadGuid", workloadGuid).Msgf("Failed to get workload status from New Relic. Full response %+v", string(responseBody))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Error().Int("code", response.StatusCode).Err(err).Str("workloadGuid", workloadGuid).Msgf("Unexpected response %+v", string(responseBody))
		return nil, errors.New("unexpected response code")
	}

	var result types.WorkloadSearchResponse
	if responseBody != nil {
		err = json.Unmarshal(responseBody, &result)
		if err != nil {
			log.Error().Err(err).Str("body", string(responseBody)).Msgf("Failed to parse body")
			return nil, err
		}
		if result.Data.Actor.Account.Workload.Collection != nil && result.Data.Actor.Account.Workload.Collection.Status != nil {
			return &result.Data.Actor.Account.Workload.Collection.Status.Value, err
		}
		log.Error().Err(err).Msgf("Unexpected response body %+v", string(responseBody))
		return nil, errors.New("unexpected response body")
	} else {
		log.Error().Err(err).Msgf("Empty response body")
		return nil, errors.New("empty response body")
	}
}

func (s *Specification) do(url string, method string, body []byte) ([]byte, *http.Response, error) {
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
	request.Header.Set("API-Key", s.ApiKey)

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
