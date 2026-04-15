package repository

func ledgerAggregateModelID(aggregateType, aggregateID string) string {
	return aggregateType + ":" + aggregateID
}
