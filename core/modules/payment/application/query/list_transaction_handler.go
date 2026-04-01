package query

import (
	"context"
	"errors"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"
)

type listTransactionHandler struct {
	paymentAccountProjectionRepo repos.PaymentAccountProjectionRepository
	paymentHistoryRepository     repos.PaymentHistoryRepository
}

func NewListTransactionHandler(repos repos.Repos) ListTransactionHandler {
	return &listTransactionHandler{
		paymentAccountProjectionRepo: repos.PaymentAccountProjectionRepository(),
		paymentHistoryRepository:     repos.PaymentHistoryRepository(),
	}
}

func (l *listTransactionHandler) Handle(ctx context.Context, req *in.ListTransactionRequest) (*out.ListTransactionResponse, error) {
	account := ctx.Value("account")
	if account == nil {
		return nil, stackerr.Error(errors.New("account not found"))
	}
	payload, ok := account.(*xpaseto.PasetoPayload)
	if !ok {
		return nil, stackerr.Error(errors.New("invalid account payload"))
	}
	options := utils.QueryOptions{
		Conditions: []utils.Condition{
			{
				Field:    "(sender_id = ? OR receiver_id = ?)",
				Value:    []interface{}{payload.AccountID, payload.AccountID},
				Operator: utils.Raw,
			},
		},
		OrderBy:        "created_at",
		OrderDirection: "DESC",
	}
	histories, err := l.paymentHistoryRepository.ListPaymentHistory(ctx, options)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	records := make([]out.TransactionRecord, 0, len(histories))
	for _, history := range histories {
		if history == nil {
			continue
		}
		records = append(records, out.TransactionRecord{
			Type:       history.Type,
			Amount:     history.Amount,
			Balance:    history.Balance,
			Date:       history.CreatedAt,
			Sender:     history.SenderName,
			SenderID:   history.SenderID,
			Receiver:   history.ReceiverName,
			ReceiverID: history.ReceiverID,
		})
	}

	return &out.ListTransactionResponse{
		Page:   req.Page,
		Limit:  req.Limit,
		Record: records,
	}, nil
}
