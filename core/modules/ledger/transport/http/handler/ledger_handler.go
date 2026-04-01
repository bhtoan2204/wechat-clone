package handler

import (
	"net/http"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/service"

	"github.com/gin-gonic/gin"
)

type LedgerHandler struct {
	service *service.LedgerService
}

func NewLedgerHandler(service *service.LedgerService) *LedgerHandler {
	return &LedgerHandler{service: service}
}

func (h *LedgerHandler) CreateTransaction(c *gin.Context) {
	var request ledgerin.CreateTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.CreateTransaction(c.Request.Context(), &request)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *LedgerHandler) GetAccountBalance(c *gin.Context) {
	response, err := h.service.GetAccountBalance(c.Request.Context(), c.Param("account_id"))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *LedgerHandler) GetTransaction(c *gin.Context) {
	response, err := h.service.GetTransaction(c.Request.Context(), c.Param("transaction_id"))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
