package apperrors

import "errors"

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrUserNotFound = errors.New("user not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidOrderStatus = errors.New("invalid order status")
	ErrOrderNotFound = errors.New("order not found")
)