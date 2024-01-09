package types

type Workload struct {
	Guid      string          `json:"guid"`
	Name      string          `json:"name"`
	Permalink string          `json:"permalink"`
	Status    *WorkloadStatus `json:"status"`
}

type WorkloadSearchResponse struct {
	Data WorkloadSearchResponseData `json:"data"`
}
type WorkloadSearchResponseData struct {
	Actor WorkloadSearchResponseActor `json:"actor"`
}
type WorkloadSearchResponseActor struct {
	Account WorkloadSearchResponseAccount `json:"account"`
}
type WorkloadSearchResponseAccount struct {
	Workload WorkloadSearchResponseWorkload `json:"workload"`
}
type WorkloadSearchResponseWorkload struct {
	Collections []Workload `json:"collections"`
	Collection  *Workload  `json:"collection"`
}
type WorkloadStatus struct {
	Value string `json:"value"`
}
