package messaging

import (
	"context"
	"fmt"

	"wechat-clone/core/shared/pkg/contxt"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func WrapConsumerCallback(consumer Consumer, failureMessage string) CallBack {
	return func(ctx context.Context, topic string, vals []byte) (err error) {
		ctx = contxt.SetRequestID(ctx)

		logger := logging.FromContext(ctx)
		if reqID := contxt.RequestIDFromCtx(ctx); reqID != "" {
			logger = logger.With("request_id", reqID)
		}
		ctx = logging.WithLogger(ctx, logger)

		defer func() {
			if recovered := recover(); recovered != nil {
				err = stackErr.Error(fmt.Errorf("panic recovered: %v", recovered))
			}
		}()

		handler := consumer.GetHandler()
		if handler == nil {
			return stackErr.Error(fmt.Errorf("consumer handler is nil"))
		}

		if err := handler(ctx, vals); err != nil {
			logger.Errorw(failureMessage, zap.Error(err), zap.String("topic", topic), zap.String("vals", string(vals)))
			return stackErr.Error(err)
		}

		return nil
	}
}
