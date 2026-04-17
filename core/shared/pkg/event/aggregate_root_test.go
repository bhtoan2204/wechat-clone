package event

import (
	"errors"
	"testing"
)

type testAggregate struct {
	AggregateRoot
	values []string
}

type testEvent struct {
	Value string
}

func newTestAggregate(id string) *testAggregate {
	agg := &testAggregate{}
	_ = InitAggregate(&agg.AggregateRoot, agg, id)
	return agg
}

func (a *testAggregate) RegisterEvents(RegisterEventsFunc) error {
	return nil
}

func (a *testAggregate) Transition(e Event) error {
	payload, ok := e.EventData.(*testEvent)
	if !ok {
		return errors.New("unexpected payload")
	}
	if payload.Value == "boom" {
		return errors.New("boom")
	}
	a.values = append(a.values, payload.Value)
	return nil
}

func TestAggregateRootApplyChangeIncrementsUnsavedVersions(t *testing.T) {
	agg := newTestAggregate("agg-1")

	if err := agg.ApplyChange(agg, &testEvent{Value: "a"}); err != nil {
		t.Fatalf("first apply failed: %v", err)
	}
	if err := agg.ApplyChange(agg, &testEvent{Value: "b"}); err != nil {
		t.Fatalf("second apply failed: %v", err)
	}

	events := agg.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Version != 1 || events[1].Version != 2 {
		t.Fatalf("expected versions [1 2], got [%d %d]", events[0].Version, events[1].Version)
	}
}

func TestAggregateRootLoadFromHistoryReturnsTransitionError(t *testing.T) {
	agg := newTestAggregate("agg-1")

	err := agg.LoadFromHistory(agg, []Event{
		{AggregateID: "agg-1", Version: 1, EventData: &testEvent{Value: "ok"}},
		{AggregateID: "agg-1", Version: 2, EventData: &testEvent{Value: "boom"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if agg.Version() != 1 {
		t.Fatalf("expected version 1 after failure, got %d", agg.Version())
	}
}
