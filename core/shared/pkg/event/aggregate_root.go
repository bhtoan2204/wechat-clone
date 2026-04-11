package event

import (
	"errors"
	"fmt"
	"go-socket/core/shared/pkg/stackErr"
	"reflect"
	"time"
)

var (
	ErrAggExisted = errors.New("can't not set ID to aggregate already got an ID")
	ErrIDEmpty    = errors.New("aggregate id can not be empty")
)

type AggregateRoot struct {
	aggregateID   string
	aggregateType string

	version int

	baseVersion int
	events      []Event
}

func (ar *AggregateRoot) SetID(id string) error {
	if id == "" {
		return stackErr.Error(ErrIDEmpty)
	}

	if id == ar.aggregateID {
		return stackErr.Error(ErrAggExisted)
	}

	ar.aggregateID = id
	return nil
}

func (ar *AggregateRoot) SetAggregateType(typ string) {
	ar.aggregateType = typ
}

func (ar *AggregateRoot) AggregateID() string {
	return ar.aggregateID
}

func (ar *AggregateRoot) AggregateType() string {
	return ar.aggregateType
}

func (ar *AggregateRoot) Root() *AggregateRoot {
	return ar
}

func (ar *AggregateRoot) Version() int {
	if len(ar.events) > 0 {
		return ar.events[len(ar.events)-1].Version
	}

	return ar.version
}

func (ar *AggregateRoot) BaseVersion() int {
	return ar.baseVersion
}

func (ar *AggregateRoot) Events() []Event {
	return ar.events
}

func (ar *AggregateRoot) CloneEvents() []Event {
	evs := make([]Event, len(ar.events))
	copy(evs, ar.events)
	return evs
}

func (ar *AggregateRoot) IsUnsaved() bool {
	return len(ar.events) > 0
}

func (ar *AggregateRoot) ApplyChange(agg Aggregate, data interface{}) error {
	return ar.ApplyChangeWithMetadata(agg, data)
}

func (ar *AggregateRoot) ApplyChangeWithMetadata(agg Aggregate, data interface{}) error {
	if ar.aggregateID == "" {
		return stackErr.Error(fmt.Errorf("missing aggregate_id, aggregate_type=%s", ar.aggregateType))
	}

	dataType := reflect.TypeOf(data)
	if dataType == nil {
		return stackErr.Error(errors.New("event data can not be nil"))
	}
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
	}
	eventType := dataType.Name()
	if eventType == "" {
		return stackErr.Error(errors.New("event name can not be empty"))
	}
	event := Event{
		AggregateID:   ar.aggregateID,
		AggregateType: ar.aggregateType,
		Version:       ar.nextVersion(),
		EventName:     eventType,
		CreatedAt:     time.Now().Unix(),
		EventData:     data,
	}

	ar.events = append(ar.events, event)
	return stackErr.Error(agg.Transition(event))
}

func (ar *AggregateRoot) LoadFromHistory(agg Aggregate, events []Event) error {
	for _, e := range events {
		if err := agg.Transition(e); err != nil {
			return stackErr.Error(err)
		}
		ar.aggregateID = e.AggregateID
		ar.version = e.Version
		ar.baseVersion = e.Version
	}
	return nil
}

func (ar *AggregateRoot) Update() {
	if len(ar.events) > 0 {
		lastEvent := ar.events[len(ar.events)-1]
		ar.version = lastEvent.Version
		ar.baseVersion = lastEvent.Version
		ar.events = []Event{}
	}
}

func (ar *AggregateRoot) SetInternal(id string, baseVersion, version int) {
	ar.aggregateID = id
	ar.baseVersion = baseVersion
	ar.version = version
	ar.events = []Event{}
}

func (ar *AggregateRoot) nextVersion() int {
	return ar.Version() + 1
}
