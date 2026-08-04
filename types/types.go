package types

import (
	"fmt"
	"strings"
)

type EventType string

const (
	EventTypeExperimentStarted EventType = "ExperimentStarted"
	EventTypeExperimentEnded   EventType = "ExperimentEnded"
	EventTypeAttackStarted     EventType = "AttackStarted"
	EventTypeAttackEnded       EventType = "AttackEnded"
)

type EventIngest struct {
	EventType         EventType `json:"eventType,omitempty"`
	ExperimentKey     string    `json:"experimentKey,omitempty"`
	ExperimentName    string    `json:"experimentName,omitempty"`
	ExecutionId       string    `json:"executionId,omitempty"`
	State             string    `json:"state,omitempty"`
	TeamName          string    `json:"teamName,omitempty"`
	TeamKey           string    `json:"teamKey,omitempty"`
	EnvironmentName   string    `json:"environmentName,omitempty"`
	PrincipalType     string    `json:"principalType,omitempty"`
	PrincipalName     string    `json:"principalName,omitempty"`
	PrincipalUsername string    `json:"principalUsername,omitempty"`
	ActionId          string    `json:"actionId,omitempty"`
	ActionName        string    `json:"actionName,omitempty"`
	ActionCustomLabel string    `json:"actionCustomLabel,omitempty"`
	Target            string    `json:"target,omitempty"`
	TargetType        string    `json:"targetType,omitempty"`
	TargetState       string    `json:"targetState,omitempty"`
}

type Workload struct {
	Guid      string          `json:"guid"`
	Name      string          `json:"name"`
	Permalink string          `json:"permalink"`
	Status    *WorkloadStatus `json:"status"`
}

type GraphQlResponse struct {
	Data   *GraphQlResponseData    `json:"data"`
	Errors *[]GraphQlResponseError `json:"errors"`
}
type GraphQlResponseData struct {
	Actor                  *GraphQlResponseActor                  `json:"actor"`
	AlertsMutingRuleCreate *GraphQlResponseAlertsMutingRuleCreate `json:"alertsMutingRuleCreate"`
}
type GraphQlResponseAlertsMutingRuleCreate struct {
	Id string `json:"id"`
}
type GraphQlResponseActor struct {
	Account  *GraphQlResponseAccount   `json:"account"`
	Accounts []GraphQlResponseAccounts `json:"accounts"`
	Entities []GraphQlResponseEntities `json:"entities"`
}
type GraphQlResponseAccount struct {
	Workload *WorkloadResponse `json:"workload"`
	AiIssues *AiIssuesResponse `json:"aiIssues"`
}
type WorkloadResponse struct {
	Collections []Workload `json:"collections"`
	Collection  *Workload  `json:"collection"`
}
type WorkloadStatus struct {
	Value string `json:"value"`
}
type GraphQlResponseAccounts struct {
	Id int64 `json:"id"`
}

type AiIssuesResponse struct {
	Incidents *IncidentsResponse `json:"incidents"`
}
type IncidentsResponse struct {
	Incidents []Incident `json:"incidents"`
}

type Incident struct {
	IncidentId  string   `json:"incidentId"`
	EntityGuids string   `json:"entityGuids"`
	EntityNames string   `json:"entityNames"`
	Priority    string   `json:"priority"`
	Title       string   `json:"title"`
	Description []string `json:"description"`
}

type GraphQlResponseEntities struct {
	Tags []GraphQlResponseTags `json:"tags"`
}

type GraphQlResponseTags struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type GraphQlResponseError struct {
	Message string `json:"message"`
	// Path is the field the error refers to. Elements are either field names or list
	// indices, hence []any.
	Path       []any                           `json:"path"`
	Extensions *GraphQlResponseErrorExtensions `json:"extensions"`
}

type GraphQlResponseErrorExtensions struct {
	ErrorClass string `json:"errorClass"`
	Code       string `json:"code"`
}

func (e *GraphQlResponseError) Error() string {
	return e.Message
}

// String renders the error including the field path and the error class, e.g.
// `actor.account.workload: user's role doesn't permit this action (UNAUTHORIZED)`.
// The message alone doesn't say which part of the query New Relic rejected.
func (e *GraphQlResponseError) String() string {
	var sb strings.Builder
	for i, segment := range e.Path {
		if i > 0 {
			sb.WriteString(".")
		}
		_, _ = fmt.Fprint(&sb, segment)
	}
	if sb.Len() > 0 {
		sb.WriteString(": ")
	}
	sb.WriteString(e.Message)
	if e.Extensions != nil && e.Extensions.ErrorClass != "" {
		sb.WriteString(" (")
		sb.WriteString(e.Extensions.ErrorClass)
		sb.WriteString(")")
	}
	return sb.String()
}
