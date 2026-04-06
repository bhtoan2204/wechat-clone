// CODE_GENERATOR: response
package out

type CreatePaymentResponse struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transaction_id"`
	ExternalRef   string `json:"external_ref"`
	Status        string `json:"status"`
	CheckoutURL   string `json:"checkout_url"`
}
