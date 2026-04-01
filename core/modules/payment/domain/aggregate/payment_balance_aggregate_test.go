package aggregate

import (
	"errors"
	"testing"
	"time"
)

func TestPaymentBalanceAggregateDepositAndWithdraw(t *testing.T) {
	agg, err := NewPaymentBalanceAggregate("account-1")
	if err != nil {
		t.Fatalf("new aggregate failed: %v", err)
	}

	now := time.Now().UTC()
	if err := agg.Deposit("tx-deposit", 100, now); err != nil {
		t.Fatalf("deposit failed: %v", err)
	}
	agg.Update()

	if err := agg.Withdraw("tx-withdraw", 40, now.Add(time.Minute)); err != nil {
		t.Fatalf("withdraw failed: %v", err)
	}

	if agg.Balance != 60 {
		t.Fatalf("unexpected balance: got %d want %d", agg.Balance, 60)
	}
}

func TestPaymentBalanceAggregateWithdrawInsufficientBalance(t *testing.T) {
	agg, err := NewPaymentBalanceAggregate("account-1")
	if err != nil {
		t.Fatalf("new aggregate failed: %v", err)
	}

	err = agg.Withdraw("tx-withdraw", 10, time.Now().UTC())
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaymentBalanceAggregateTransferAndReceive(t *testing.T) {
	sender, err := NewPaymentBalanceAggregate("sender-account")
	if err != nil {
		t.Fatalf("new sender aggregate failed: %v", err)
	}

	receiver, err := NewPaymentBalanceAggregate("receiver-account")
	if err != nil {
		t.Fatalf("new receiver aggregate failed: %v", err)
	}

	now := time.Now().UTC()
	if err := sender.Deposit("seed", 200, now); err != nil {
		t.Fatalf("seed deposit failed: %v", err)
	}
	sender.Update()

	if err := sender.Transfer("tx-transfer", 75, receiver.AccountID, now.Add(time.Minute)); err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	if err := receiver.Receive("tx-transfer", 75, sender.AccountID, now.Add(time.Minute)); err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	if sender.Balance != 125 {
		t.Fatalf("unexpected sender balance: got %d want %d", sender.Balance, 125)
	}
	if receiver.Balance != 75 {
		t.Fatalf("unexpected receiver balance: got %d want %d", receiver.Balance, 75)
	}
}
