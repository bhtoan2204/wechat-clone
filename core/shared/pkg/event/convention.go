package event

import (
	"errors"
	"reflect"
	"strings"
)

var (
	ErrAggregateRootRequired = errors.New("aggregate root is required")
	ErrAggregateTypeEmpty    = errors.New("aggregate type can not be empty")
	ErrEventNameEmpty        = errors.New("event name can not be empty")
	ErrUnsupportedEventType  = errors.New("unsupported event type")
)

func AggregateTypeName(aggregate any) string {
	return typeName(aggregate)
}

func EventName(payload any) string {
	return typeName(payload)
}

func InitAggregate(root *AggregateRoot, aggregate any, aggregateID string) error {
	if root == nil {
		return ErrAggregateRootRequired
	}

	aggregateType := AggregateTypeName(aggregate)
	if aggregateType == "" {
		return ErrAggregateTypeEmpty
	}

	root.SetAggregateType(aggregateType)
	return root.SetID(strings.TrimSpace(aggregateID))
}

func typeName(value any) string {
	typ := reflect.TypeOf(value)
	if typ == nil {
		return ""
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}
