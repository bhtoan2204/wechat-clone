package eventstore

import (
	"context"
	"go-socket/core/modules/payment/infra/persistent/model"
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type aggregateRepo struct {
	db         *gorm.DB
	serializer event.Serializer
}

func NewAggregateStore(db *gorm.DB, eventSerializer event.Serializer) *aggregateRepo {
	return &aggregateRepo{
		db:         db,
		serializer: eventSerializer,
	}
}

func (r aggregateRepo) Versioning(ctx context.Context, agg event.Aggregate) (int, error) {
	log := logging.FromContext(ctx).Named("Versioning")
	aggregateID := agg.Root().AggregateID()
	expectedVersion := agg.Root().BaseVersion()
	newVersion := agg.Root().Version()

	if err := r.db.Raw(updateVersionSQL, newVersion, aggregateID, expectedVersion).Error; err != nil {
		log.Errorw("Failed to update version", zap.Error(err), zap.Int("expectedVersion", expectedVersion), zap.Int("newVersion", newVersion))
		return 0, stackerr.Error(err)
	}
	return newVersion, nil
}

func (r aggregateRepo) CreateSnapshot(ctx context.Context, aggregate event.Aggregate) error {
	data, err := r.serializer.Marshal(aggregate.Root())
	if err != nil {
		return err
	}
	return r.db.Create(&model.PaymentBalanceSnapshotModel{
		AggregateID: aggregate.Root().AggregateID(),
		Version:     aggregate.Root().Version(),
		State:       string(data),
	}).Error
}

func (r aggregateRepo) GetSnapshot(ctx context.Context, aggregateID string, version int, agg event.Aggregate) (event.Aggregate, error) {
	events := []model.PaymentEventModel{}
	if err := r.db.Raw(readSnapshotSQL, aggregateID, version).Scan(&events).Error; err != nil {
		return nil, stackerr.Error(err)
	}
	root := agg.Root()
	for _, snapshot := range events {
		if err := r.serializer.Unmarshal([]byte(snapshot.EventData), root); err != nil {
			return nil, stackerr.Error(err)
		}
	}
	return agg, nil
}
