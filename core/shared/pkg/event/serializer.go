package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-socket/core/shared/pkg/stackErr"
)

type eventFunc = func() interface{}
type RegisterEventsFunc = func(events ...interface{}) error

var _ Serializer = (*serializer)(nil)

//go:generate mockgen -package=event -destination=serializer_mock.go -source=serializer.go
type Serializer interface {
	ToEventsFunc(events ...interface{}) []eventFunc
	RegisterAggregate(agg BaseAggregate) error
	Register(agg Aggregate, eventsFunc []eventFunc) error
	RegisterTypes(agg Aggregate, eventsFunc ...eventFunc) error
	Type(typ, name string) (eventFunc, bool)
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type serializer struct {
	eventRegister map[string]eventFunc
}

func NewSerializer() Serializer {
	return &serializer{
		eventRegister: make(map[string]func() interface{}),
	}
}

func (s *serializer) ToEventsFunc(events ...interface{}) []eventFunc {
	res := []eventFunc{}
	for _, item := range events {
		res = append(res, eventToFunc(item))
	}

	return res
}

func (s *serializer) RegisterAggregate(agg BaseAggregate) error {
	typ := AggregateTypeName(agg)
	if typ == "" {
		return stackErr.Error(fmt.Errorf("not found aggregate"))
	}

	fu := func(events ...interface{}) error {
		listF := s.ToEventsFunc(events...)
		for _, f := range listF {
			event := f()
			eName := EventName(event)
			if eName == "" {
				return stackErr.Error(errors.New("name of event is missing"))
			}

			s.eventRegister[typ+"_"+eName] = f
		}

		return nil
	}

	return agg.RegisterEvents(fu)
}

func (s *serializer) Register(agg Aggregate, eventsFunc []eventFunc) error {
	typ := AggregateTypeName(agg)
	if typ == "" {
		return stackErr.Error(errors.New("not found aggregate"))
	}

	for _, f := range eventsFunc {
		event := f()
		eName := EventName(event)
		if eName == "" {
			return stackErr.Error(errors.New("event name is missing"))
		}
		s.eventRegister[typ+"_"+eName] = f
	}

	return nil
}

func (s *serializer) RegisterTypes(agg Aggregate, eventsFunc ...eventFunc) error {
	return s.Register(agg, eventsFunc)
}

func (s *serializer) Type(typ, name string) (eventFunc, bool) {
	f, ok := s.eventRegister[typ+"_"+name]
	return f, ok
}

func (s *serializer) Marshal(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return data, nil
}

func (s *serializer) Unmarshal(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func eventToFunc(e interface{}) eventFunc {
	return func() interface{} {
		return e
	}
}
