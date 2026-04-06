// CODE_GENERATOR: request

package in

type ListTransactionRequest struct {
	Page  int64 `json:"page" form:"page"`
	Limit int64 `json:"limit" form:"limit"`
}

func (r *ListTransactionRequest) Validate() error {
	return nil
}
