// CODE_GENERATOR: response
package out

type ListTransactionResponse struct {
	Page    int64               `json:"page"`
	Limit   int64               `json:"limit"`
	Balance int64               `json:"balance"`
	Records []TransactionRecord `json:"records"`
}

type TransactionRecord struct {
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	Balance    int64  `json:"balance"`
	Date       string `json:"date"`
	Sender     string `json:"sender"`
	SenderID   string `json:"sender_id"`
	Receiver   string `json:"receiver"`
	ReceiverID string `json:"receiver_id"`
}
