package command

import (
	"context"
	"time"

	"wechat-clone/core/modules/relationship/support"
	"wechat-clone/core/shared/pkg/stackErr"
)

func currentAccountID(ctx context.Context) (string, error) {
	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return accountID, nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
