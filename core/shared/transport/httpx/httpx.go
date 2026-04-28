package httpx

import (
	"errors"
	"net/http"

	"wechat-clone/core/shared/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func Wrap(h interface {
	Handle(c *gin.Context) (interface{}, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "handler is nil"})
			return
		}
		data, err := h.Handle(c)
		if c.Writer.Written() {
			return
		}
		if err != nil {
			var appError *apperr.Error
			if errors.As(err, &appError) {
				c.JSON(appError.HTTPStatus(), gin.H{"code": appError.Code(), "message": appError.Message()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": "internal_error", "message": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, data)
	}
}
