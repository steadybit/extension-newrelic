package types

type Workload struct {
	Guid      string          `json:"guid"`
	Name      string          `json:"name"`
	Permalink string          `json:"permalink"`
	Status    *WorkloadStatus `json:"status"`
}

type GraphQlResponse struct {
	Data GraphQlResponseData `json:"data"`
}
type GraphQlResponseData struct {
	Actor GraphQlResponseActor `json:"actor"`
}
type GraphQlResponseActor struct {
	Account  GraphQlResponseAccount    `json:"account"`
	Accounts []GraphQlResponseAccounts `json:"accounts"`
}
type GraphQlResponseAccount struct {
	Workload WorkloadResponse `json:"workload"`
}
type WorkloadResponse struct {
	Collections []Workload `json:"collections"`
	Collection  *Workload  `json:"collection"`
}
type WorkloadStatus struct {
	Value string `json:"value"`
}
type GraphQlResponseAccounts struct {
	Id string `json:"id"`
}
