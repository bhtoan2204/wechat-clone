// CODE_GENERATOR: response
package out

type DepositResponse struct {
	Message       string `json:"message"`
	TransactionID string `json:"transaction_id"`
	Balance       int64  `json:"balance"`
	Version       int    `json:"version"`
}
