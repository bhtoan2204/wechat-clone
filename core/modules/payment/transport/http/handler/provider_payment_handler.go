package handler

import (
	"errors"
	"io"
	"net/http"

	paymentin "go-socket/core/modules/payment/application/dto/in"
	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/modules/payment/providers"

	"github.com/gin-gonic/gin"
)

type ProviderPaymentHandler struct {
	service *paymentservice.PaymentService
}

func NewProviderPaymentHandler(service *paymentservice.PaymentService) *ProviderPaymentHandler {
	return &ProviderPaymentHandler{service: service}
}

func (h *ProviderPaymentHandler) CreatePayment(c *gin.Context) {
	var request paymentin.CreatePaymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := accountIDFromContext(c.Request.Context())
	if err == nil {
		if request.DebitAccountID == "" {
			request.DebitAccountID = accountID
		} else if request.DebitAccountID != accountID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "debit_account_id must match authenticated account"})
			return
		}
	}

	response, err := h.service.CreatePayment(c.Request.Context(), &request)
	if err != nil {
		writeProviderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *ProviderPaymentHandler) HandleWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to read request body"})
		return
	}

	response, err := h.service.HandleWebhook(
		c.Request.Context(),
		c.Param("provider"),
		payload,
		c.GetHeader("X-Signature"),
	)
	if err != nil {
		writeProviderError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func writeProviderError(c *gin.Context, err error) {
	status := http.StatusInternalServerError

	switch {
	case isProviderValidation(err):
		status = http.StatusBadRequest
	case isProviderDuplicate(err):
		status = http.StatusConflict
	case isProviderNotFound(err):
		status = http.StatusNotFound
	case isUnknownProvider(err):
		status = http.StatusBadRequest
	case errors.Is(err, providers.ErrInvalidWebhookSignature):
		status = http.StatusUnauthorized
	}

	c.JSON(status, gin.H{"error": err.Error()})
}

func isProviderValidation(err error) bool {
	return errors.Is(err, paymentservice.ErrValidation)
}

func isProviderDuplicate(err error) bool {
	return errors.Is(err, paymentservice.ErrDuplicateTransaction) || errors.Is(err, paymentservice.ErrDuplicatePayment)
}

func isProviderNotFound(err error) bool {
	return errors.Is(err, paymentservice.ErrTransactionNotFound) || errors.Is(err, paymentservice.ErrPaymentIntentNotFound)
}

func isUnknownProvider(err error) bool {
	return errors.Is(err, providers.ErrProviderNotFound)
}
