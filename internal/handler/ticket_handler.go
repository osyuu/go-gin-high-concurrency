package handler

import (
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/service"
	apperrors "go-gin-high-concurrency/pkg/app_errors"
	"go-gin-high-concurrency/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TicketHandler struct {
	service service.TicketService
}

func NewTicketHandler(service service.TicketService) *TicketHandler {
	return &TicketHandler{service: service}
}

func (h *TicketHandler) RegisterRoutes(r *gin.Engine) {
	router := r.Group("/api/v1")
	{
		router.GET("tickets", h.List)
		router.GET("tickets/:uuid", h.GetByTicketID)
		router.POST("tickets", h.Create)
		router.PUT("tickets/:uuid", h.UpdateByTicketID)
		router.DELETE("tickets/:uuid", h.DeleteByTicketID)
	}
}

// CreateTicketRequest 建立票券請求
type CreateTicketRequest struct {
	EventID    int     `json:"event_id" binding:"required"`
	Name       string  `json:"name" binding:"required"`
	Price      float64 `json:"price" binding:"required"`
	TotalStock int     `json:"total_stock" binding:"required"`
	MaxPerUser int     `json:"max_per_user" binding:"required"`
}

// UpdateTicketRequest 更新票券請求
type UpdateTicketRequest struct {
	Name       *string  `json:"name"`
	Price      *float64 `json:"price"`
	MaxPerUser *int     `json:"max_per_user"`
}

func (h *TicketHandler) List(c *gin.Context) {
	tickets, err := h.service.List(c)
	if err != nil {
		h.handleError(c, err, "List")
		return
	}
	c.JSON(http.StatusOK, tickets)
}

func (h *TicketHandler) GetByTicketID(c *gin.Context) {
	uuidStr := c.Param("uuid")
	ticketID, err := uuid.Parse(uuidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket uuid"})
		return
	}
	ticket, err := h.service.GetByTicketID(c, ticketID)
	if err != nil {
		h.handleError(c, err, "GetByTicketID")
		return
	}
	c.JSON(http.StatusOK, ticket)
}

func (h *TicketHandler) Create(c *gin.Context) {
	var req CreateTicketRequest
	if err := BindJson(c, &req); err != nil {
		return
	}
	ticket := &model.Ticket{
		EventID:        req.EventID,
		Name:           req.Name,
		Price:          req.Price,
		TotalStock:     req.TotalStock,
		RemainingStock: req.TotalStock,
		MaxPerUser:     req.MaxPerUser,
	}
	created, err := h.service.Create(c, ticket)
	if err != nil {
		h.handleError(c, err, "Create")
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *TicketHandler) UpdateByTicketID(c *gin.Context) {
	uuidStr := c.Param("uuid")
	ticketID, err := uuid.Parse(uuidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket uuid"})
		return
	}
	var req UpdateTicketRequest
	if err := BindJson(c, &req); err != nil {
		return
	}
	if req.Name == nil && req.Price == nil && req.MaxPerUser == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one of name, price or max_per_user is required"})
		return
	}
	params := model.UpdateTicketParams{
		Name:       req.Name,
		Price:      req.Price,
		MaxPerUser: req.MaxPerUser,
	}
	updated, err := h.service.UpdateByTicketID(c, ticketID, params)
	if err != nil {
		h.handleError(c, err, "UpdateByTicketID")
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *TicketHandler) DeleteByTicketID(c *gin.Context) {
	uuidStr := c.Param("uuid")
	ticketID, err := uuid.Parse(uuidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket uuid"})
		return
	}
	err = h.service.DeleteByTicketID(c, ticketID)
	if err != nil {
		h.handleError(c, err, "DeleteByTicketID")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TicketHandler) handleError(c *gin.Context, err error, operation string) {
	log := logger.WithComponent("handler").With(zap.String("operation", operation), zap.Error(err))
	switch {
	case err == apperrors.ErrTicketNotFound:
		log.Warn("Ticket not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
	case err == apperrors.ErrInvalidInput:
		log.Warn("Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
	default:
		log.Error("Unexpected error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}
