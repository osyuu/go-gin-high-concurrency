package handler

import (
	"go-gin-high-concurrency/internal/handler"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/service/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupOrderTestRouter(mockService *mocks.MockOrderService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 使用 NewOrderHandler 注入 mock service ✅
	orderHandler := handler.NewOrderHandler(mockService)

	router.GET("/api/v1/orders", orderHandler.GetOrders)
	router.GET("/api/v1/orders/:uuid", orderHandler.GetOrder)
	router.POST("/api/v1/orders", orderHandler.CreateOrder)
	router.PUT("/api/v1/orders/:uuid/confirm", orderHandler.ConfirmOrder)
	router.PUT("/api/v1/orders/:uuid/cancel", orderHandler.CancelOrder)
	router.DELETE("/api/v1/orders/:uuid", orderHandler.DeleteOrder)

	return router
}

func TestCreateOrder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().PrepareOrder(mock.Anything, mock.Anything).Return(&model.Order{
			ID:         1,
			UserID:     1,
			TicketID:   1,
			Quantity:   1,
			TotalPrice: 100,
			Status:     "pending",
		}, nil).Once()

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

	t.Run("Failed - ErrInsufficientStock", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().PrepareOrder(mock.Anything, mock.Anything).Return(nil, apperrors.ErrInsufficientStock).Once()

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

	t.Run("Failed - ErrInternalServerError", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().PrepareOrder(mock.Anything, mock.Anything).Return(nil, apperrors.ErrInternalServerError).Once()

		createOrderRequest := model.CreateOrderRequest{
			UserID:   1,
			TicketID: 1,
			Quantity: 1,
		}
		req := createJSONHTTPRequest("POST", "/api/v1/orders", createOrderRequest)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Failed - BindingError", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		req := createJSONHTTPRequest("POST", "/api/v1/orders", InvalidJSON)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "PrepareOrder")
	})
}

func TestGetOrder(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	t.Run("Success", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().GetOrderByOrderID(mock.Anything, mock.Anything).Return(&model.Order{
			UserID:     1,
			TicketID:   1,
			Quantity:   2,
			TotalPrice: 2000,
			Status:     model.OrderStatusPending,
		}, nil).Once()

		req := httptest.NewRequest("GET", "/api/v1/orders/"+validUUID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		req := httptest.NewRequest("GET", "/api/v1/orders/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "GetOrderByOrderID")
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		notFoundUUID := "550e8400-e29b-41d4-a716-446655440099"
		mockService.EXPECT().GetOrderByOrderID(mock.Anything, mock.Anything).Return(nil, apperrors.ErrOrderNotFound).Once()

		req := httptest.NewRequest("GET", "/api/v1/orders/"+notFoundUUID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetOrders(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().OrderList(mock.Anything).Return([]*model.Order{
			{ID: 1, UserID: 1, TicketID: 1, Quantity: 2, Status: model.OrderStatusPending},
			{ID: 2, UserID: 1, TicketID: 2, Quantity: 1, Status: model.OrderStatusConfirmed},
		}, nil).Once()

		// request
		req := httptest.NewRequest("GET", "/api/v1/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Failed - InternalServerError", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().OrderList(mock.Anything).Return(nil, apperrors.ErrInternalServerError).Once()

		// request
		req := httptest.NewRequest("GET", "/api/v1/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestConfirmOrder(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440010"
	t.Run("Success", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().ConfirmOrderByOrderID(mock.Anything, mock.Anything).Return(nil).Once()

		req := httptest.NewRequest("PUT", "/api/v1/orders/"+validUUID+"/confirm", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		notFoundUUID := "550e8400-e29b-41d4-a716-446655440099"
		mockService.EXPECT().ConfirmOrderByOrderID(mock.Anything, mock.Anything).Return(apperrors.ErrOrderNotFound).Once()

		req := httptest.NewRequest("PUT", "/api/v1/orders/"+notFoundUUID+"/confirm", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		req := httptest.NewRequest("PUT", "/api/v1/orders/invalid/confirm", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "ConfirmOrderByOrderID")
	})
}

func TestCancelOrder(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440020"
	t.Run("Success", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().CancelOrderByOrderID(mock.Anything, mock.Anything).Return(nil).Once()

		req := httptest.NewRequest("PUT", "/api/v1/orders/"+validUUID+"/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		notFoundUUID := "550e8400-e29b-41d4-a716-446655440099"
		mockService.EXPECT().CancelOrderByOrderID(mock.Anything, mock.Anything).Return(apperrors.ErrOrderNotFound).Once()

		req := httptest.NewRequest("PUT", "/api/v1/orders/"+notFoundUUID+"/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		req := httptest.NewRequest("PUT", "/api/v1/orders/invalid/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "CancelOrderByOrderID")
	})
}

func TestDeleteOrder(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440030"
	t.Run("Success", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		mockService.EXPECT().DeleteOrderByOrderID(mock.Anything, mock.Anything).Return(nil).Once()

		req := httptest.NewRequest("DELETE", "/api/v1/orders/"+validUUID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		notFoundUUID := "550e8400-e29b-41d4-a716-446655440099"
		mockService.EXPECT().DeleteOrderByOrderID(mock.Anything, mock.Anything).Return(apperrors.ErrOrderNotFound).Once()

		req := httptest.NewRequest("DELETE", "/api/v1/orders/"+notFoundUUID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		mockService := mocks.NewMockOrderService(t)
		router := setupOrderTestRouter(mockService)

		req := httptest.NewRequest("DELETE", "/api/v1/orders/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "DeleteOrderByOrderID")
	})
}
