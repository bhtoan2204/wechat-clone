package event

//go:generate mockgen -package=event -destination=aggregate_mock.go -source=aggregate.go
type BaseAggregate interface {
	RegisterEvents(RegisterEventsFunc) error
}

// All aggregate must implement this interface
//
//go:generate mockgen -package=event -destination=aggregate_mock.go -source=aggregate.go
type Aggregate interface {
	BaseAggregate
	Root() *AggregateRoot
	Transition(e Event) error
}
