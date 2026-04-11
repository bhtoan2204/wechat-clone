package projection

import (
	"context"
	"fmt"

	"go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/contxt"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (p *processor) processMessage(consume messaging.Consumer) messaging.CallBack {
	return func(ctx context.Context, topic string, vals []byte) (err error) {
		ctx = contxt.SetRequestID(ctx)

		logger := logging.FromContext(ctx)
		if reqID := contxt.RequestIDFromCtx(ctx); reqID != "" {
			logger = logger.With("request_id", reqID)
		}
		ctx = logging.WithLogger(ctx, logger)

		defer func() {
			if r := recover(); r != nil {
				err = stackErr.Error(fmt.Errorf("panic recovered: %v", r))
			}
		}()

		if err = consume.GetHandler()(ctx, vals); err != nil {
			logger.Errorw("Handle room projection message failed", zap.Error(err), zap.String("topic", topic), zap.String("vals", string(vals)))
			return stackErr.Error(err)
		}

		return nil
	}
}
