package types

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
}
