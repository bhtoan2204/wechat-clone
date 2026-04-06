// CODE_GENERATOR: response
package out

type ProcessWebhookResponse struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transaction_id"`
	ExternalRef   string `json:"external_ref"`
	Status        string `json:"status"`
	Duplicate     bool   `json:"duplicate"`
	LedgerPosted  bool   `json:"ledger_posted"`
}
