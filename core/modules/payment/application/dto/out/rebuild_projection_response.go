// CODE_GENERATOR: response
package out

type RebuildProjectionResponse struct {
	Message             string `json:"message"`
	Mode                string `json:"mode"`
	AccountID           string `json:"account_id"`
	Accounts            int    `json:"accounts"`
	EventsReplayed      int    `json:"events_replayed"`
	TransactionsRebuilt int    `json:"transactions_rebuilt"`
	BalancesRebuilt     int    `json:"balances_rebuilt"`
	Note                string `json:"note"`
}
