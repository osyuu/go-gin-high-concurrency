package handler

import (
	"errors"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/service"
	apperrors "go-gin-high-concurrency/pkg/app_errors"
	"go-gin-high-concurrency/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) RegisterRoutes(r *gin.Engine) {
	router := r.Group("/api/v1")
	{
		router.GET("orders", h.GetOrders)
		router.GET("orders/:id", h.GetOrder)
		router.POST("orders", h.CreateOrder)
		router.PUT("orders/:id/confirm", h.ConfirmOrder)
		router.PUT("orders/:id/cancel", h.CancelOrder)
		router.DELETE("orders/:id", h.DeleteOrder)
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var orderReq model.CreateOrderRequest

	if err := BindJson(c, &orderReq); err != nil {
		return
	}

	created, err := h.service.PrepareOrder(c, orderReq)
	if err != nil {
		h.handleOrderError(c, err, "CreateOrder")
		return
	}

	h.handleOrderSuccess(c, created, http.StatusCreated)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		h.handleOrderError(c, err, "GetOrder")
		return
	}
	order, err := h.service.GetOrderByID(c, idInt)
	if err != nil {
		h.handleOrderError(c, err, "GetOrder")
		return
	}

	h.handleOrderSuccess(c, order, http.StatusOK)
}

func (h *OrderHandler) GetOrders(c *gin.Context) {
	orders, err := h.service.OrderList(c)
	if err != nil {
		h.handleOrderError(c, err, "GetOrders")
		return
	}

	h.handleOrderSuccess(c, orders, http.StatusOK)
}

func (h *OrderHandler) ConfirmOrder(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		h.handleOrderError(c, err, "ConfirmOrder")
		return
	}
	err = h.service.ConfirmOrder(c, idInt)
	if err != nil {
		h.handleOrderError(c, err, "ConfirmOrder")
		return
	}

	h.handleOrderSuccess(c, nil, http.StatusOK)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		h.handleOrderError(c, err, "CancelOrder")
		return
	}
	err = h.service.CancelOrder(c, idInt)
	if err != nil {
		h.handleOrderError(c, err, "CancelOrder")
		return
	}

	h.handleOrderSuccess(c, nil, http.StatusOK)
}

func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		h.handleOrderError(c, err, "DeleteOrder")
		return
	}
	err = h.service.DeleteOrder(c, idInt)
	if err != nil {
		h.handleOrderError(c, err, "DeleteOrder")
		return
	}

	h.handleOrderSuccess(c, nil, http.StatusOK)
}

// Helper functions

func (h *OrderHandler) handleOrderError(c *gin.Context, err error, operation string) {
	log := logger.WithComponent("handler").With(zap.String("operation", operation), zap.Error(err))
	switch {
	case errors.Is(err, apperrors.ErrInsufficientStock):
		log.Warn("Insufficient stock")
		c.JSON(http.StatusConflict, gin.H{
			"error": "Insufficient stock",
		})
	case errors.Is(err, apperrors.ErrExceedsMaxPerUser):
		log.Warn("Exceeds max per user")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Exceeds max per user",
		})
	case errors.Is(err, apperrors.ErrTicketNotFound):
		log.Warn("Ticket not found")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ticket not found",
		})
	case errors.Is(err, apperrors.ErrOrderNotFound):
		log.Warn("Order not found")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found",
		})
	default:
		log.Error("Unexpected error")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	}
}

func (h *OrderHandler) handleOrderSuccess(c *gin.Context, data interface{}, statusCode int) {
	if data != nil {
		c.JSON(statusCode, data)
	} else {
		c.Status(statusCode)
	}
}
