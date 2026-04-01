package handler

import (
	"io"
	"net/http"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/service"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var request ledgerin.CreatePaymentRequest
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
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
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
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
