// CODE_GENERATOR - do not edit: response
package out

type ProcessWebhookResponse struct {
	Provider      string                    `json:"provider,omitempty"`
	TransactionID string                    `json:"transaction_id,omitempty"`
	ExternalRef   string                    `json:"external_ref,omitempty"`
	Status        string                    `json:"status,omitempty"`
	Duplicate     bool                      `json:"duplicate,omitempty"`
	LedgerPosted  bool                      `json:"ledger_posted,omitempty"`
	Events        []PaymentIntegrationEvent `json:"events,omitempty"`
}

type PaymentIntegrationEvent struct {
	Name     string `json:"name,omitempty"`
	DataJson string `json:"data_json,omitempty"`
}
