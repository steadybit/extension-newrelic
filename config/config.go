/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
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
