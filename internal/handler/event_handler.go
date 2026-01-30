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

type EventHandler struct {
	service service.EventService
}

func NewEventHandler(service service.EventService) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) RegisterRoutes(r *gin.Engine) {
	router := r.Group("/api/v1")
	{
		router.GET("events", h.List)
		router.GET("events/:uuid", h.GetByEventID)
		router.POST("events", h.Create)
		router.PUT("events/:uuid", h.UpdateByEventID)
	}
}

// CreateEventRequest 建立活動請求
type CreateEventRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

// UpdateEventRequest 更新活動請求
type UpdateEventRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *EventHandler) List(c *gin.Context) {
	events, err := h.service.List(c)
	if err != nil {
		h.handleError(c, err, "List")
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *EventHandler) GetByEventID(c *gin.Context) {
	uuidStr := c.Param("uuid")
	eventID, err := uuid.Parse(uuidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event uuid"})
		return
	}
	event, err := h.service.GetByEventID(c, eventID)
	if err != nil {
		h.handleError(c, err, "GetByEventID")
		return
	}
	c.JSON(http.StatusOK, event)
}

func (h *EventHandler) Create(c *gin.Context) {
	var req CreateEventRequest
	if err := BindJson(c, &req); err != nil {
		return
	}
	event := &model.Event{
		Name:        req.Name,
		Description: req.Description,
	}
	created, err := h.service.Create(c, event)
	if err != nil {
		h.handleError(c, err, "Create")
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *EventHandler) UpdateByEventID(c *gin.Context) {
	uuidStr := c.Param("uuid")
	eventID, err := uuid.Parse(uuidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event uuid"})
		return
	}
	var req UpdateEventRequest
	if err := BindJson(c, &req); err != nil {
		return
	}
	if req.Name == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one of name or description is required"})
		return
	}
	params := model.UpdateEventParams{
		Name:        req.Name,
		Description: req.Description,
	}
	updated, err := h.service.UpdateByEventID(c, eventID, params)
	if err != nil {
		h.handleError(c, err, "UpdateByEventID")
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *EventHandler) handleError(c *gin.Context, err error, operation string) {
	log := logger.WithComponent("handler").With(zap.String("operation", operation), zap.Error(err))
	switch {
	case err == apperrors.ErrEventNotFound:
		log.Warn("Event not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
	case err == apperrors.ErrInvalidInput:
		log.Warn("Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
	default:
		log.Error("Unexpected error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}
