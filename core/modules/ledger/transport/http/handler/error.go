package handler

import (
	"errors"
	"net/http"

	"go-socket/core/modules/ledger/application/service"

	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, service.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrDuplicateTransaction), errors.Is(err, service.ErrDuplicatePayment):
		status = http.StatusConflict
	case errors.Is(err, service.ErrTransactionNotFound), errors.Is(err, service.ErrPaymentIntentNotFound):
		status = http.StatusNotFound
	}

	c.JSON(status, gin.H{"error": err.Error()})
}
