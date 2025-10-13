package app_errors

import "errors"

var (
	// common errors
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")

	// Ticket related errors
	ErrTicketNotFound    = errors.New("ticket not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidTicketData = errors.New("invalid ticket data")

	// Order related errors
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidOrderStatus = errors.New("invalid order status")
	ErrExceedsMaxPerUser  = errors.New("exceeds maximum tickets per user")

	// User related errors
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateEmail = errors.New("email already exists")
)
