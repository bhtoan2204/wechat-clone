package out

type CreatePaymentResponse struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transaction_id"`
	ExternalRef   string `json:"external_ref,omitempty"`
	Status        string `json:"status"`
	CheckoutURL   string `json:"checkout_url,omitempty"`
}

type ProcessWebhookResponse struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transaction_id"`
	ExternalRef   string `json:"external_ref,omitempty"`
	Status        string `json:"status"`
	Duplicate     bool   `json:"duplicate"`
	LedgerPosted  bool   `json:"ledger_posted"`
}
