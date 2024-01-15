package types

type Workload struct {
	Guid      string          `json:"guid"`
	Name      string          `json:"name"`
	Permalink string          `json:"permalink"`
	Status    *WorkloadStatus `json:"status"`
}

type GraphQlResponse struct {
	Data *GraphQlResponseData `json:"data"`
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
