package handler

import (
	"go-gin-high-concurrency/internal/handler"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/test/internal/mocks/services"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupOrderTestRouter(mockService *services.OrderServiceMock) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 使用 NewOrderHandler 注入 mock service ✅
	orderHandler := handler.NewOrderHandler(mockService)

	router.GET("/api/v1/orders", orderHandler.GetOrders)
	router.GET("/api/v1/orders/:id", orderHandler.GetOrder)
	router.POST("/api/v1/orders", orderHandler.CreateOrder)
	router.PUT("/api/v1/orders/:id/confirm", orderHandler.ConfirmOrder)
	router.PUT("/api/v1/orders/:id/cancel", orderHandler.CancelOrder)
	router.DELETE("/api/v1/orders/:id", orderHandler.DeleteOrder)

	return router
}

func TestCreateOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("Create", mock.Anything, mock.Anything).Return(&model.Order{
			ID:         1,
			UserID:     1,
			TicketID:   1,
			Quantity:   1,
			TotalPrice: 100,
			Status:     "pending",
		}, nil)

		createOrderRequest := model.CreateOrderRequest{
			UserID:   1,
			TicketID: 1,
			Quantity: 1,
		}

		// request
		req := createJSONHTTPRequest("POST", "/api/v1/orders", createOrderRequest)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("BindingError", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		req := createJSONHTTPRequest("POST", "/api/v1/orders", InvalidJSON)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("InsufficientStock", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("Create", mock.Anything, mock.Anything).Return(nil, apperrors.ErrInsufficientStock)

		createOrderRequest := model.CreateOrderRequest{
			UserID:   1,
			TicketID: 1,
			Quantity: 1,
		}

		// request
		req := createJSONHTTPRequest("POST", "/api/v1/orders", createOrderRequest)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusConflict, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("GetByID", mock.Anything, 123).Return(&model.Order{
			ID:         123,
			UserID:     1,
			TicketID:   1,
			Quantity:   2,
			TotalPrice: 2000,
			Status:     model.OrderStatusPending,
		}, nil)

		// request
		req := httptest.NewRequest("GET", "/api/v1/orders/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidID", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		// request
		req := httptest.NewRequest("GET", "/api/v1/orders/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertNotCalled(t, "GetByID")
	})
}

func TestGetOrders(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("List", mock.Anything).Return([]*model.Order{
			{ID: 1, UserID: 1, TicketID: 1, Quantity: 2, Status: model.OrderStatusPending},
			{ID: 2, UserID: 1, TicketID: 2, Quantity: 1, Status: model.OrderStatusConfirmed},
		}, nil)

		// request
		req := httptest.NewRequest("GET", "/api/v1/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestConfirmOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("Confirm", mock.Anything, 123).Return(nil)

		// request
		req := httptest.NewRequest("PUT", "/api/v1/orders/123/confirm", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("Confirm", mock.Anything, 99999).Return(apperrors.ErrOrderNotFound)

		// request
		req := httptest.NewRequest("PUT", "/api/v1/orders/99999/confirm", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestCancelOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("Cancel", mock.Anything, 123).Return(nil)

		// request
		req := httptest.NewRequest("PUT", "/api/v1/orders/123/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestDeleteOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := services.NewOrderServiceMock()
		router := setupOrderTestRouter(mockService)

		mockService.On("Delete", mock.Anything, 123).Return(nil)

		// request
		req := httptest.NewRequest("DELETE", "/api/v1/orders/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}
