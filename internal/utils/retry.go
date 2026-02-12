/*
Copyright 2025 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// ErrorType represents the category of an error
type ErrorType string

const (
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork ErrorType = "Network"
	// ErrorTypeAuth represents authentication/authorization errors
	ErrorTypeAuth ErrorType = "Auth"
	// ErrorTypeNotFound represents resource not found errors
	ErrorTypeNotFound ErrorType = "NotFound"
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "Validation"
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout ErrorType = "Timeout"
	// ErrorTypeUnknown represents unknown errors
	ErrorTypeUnknown ErrorType = "Unknown"
)

// RetryableError wraps an error with retry information
type RetryableError struct {
	error
	Type       ErrorType
	RetryAfter time.Duration
	Retryable  bool
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	return e.error.Error()
}

// Unwrap implements the error unwrapping interface
func (e *RetryableError) Unwrap() error {
	return e.error
}

// Backoff configurations for different error types
var (
	// NetworkErrorBackoff for network-related errors
	NetworkErrorBackoff = wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    5,
		Cap:      5 * time.Minute,
	}

	// AuthErrorBackoff for authentication errors
	AuthErrorBackoff = wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    3,
		Cap:      15 * time.Minute,
	}

	// ValidationErrorBackoff for validation errors (rarely retry)
	ValidationErrorBackoff = wait.Backoff{
		Duration: 5 * time.Minute,
		Factor:   1.5,
		Steps:    2,
		Cap:      10 * time.Minute,
	}

	// DefaultBackoff for unknown errors
	DefaultBackoff = wait.Backoff{
		Duration: 30 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    5,
		Cap:      10 * time.Minute,
	}
)

// ClassifyError analyzes an error and returns its type and retry strategy
func ClassifyError(err error) *RetryableError {
	if err == nil {
		return nil
	}

	errorType := classifyErrorType(err)
	backoff := getBackoffForErrorType(errorType)
	retryable := isRetryable(errorType)

	return &RetryableError{
		error:      err,
		Type:       errorType,
		RetryAfter: backoff.Duration,
		Retryable:  retryable,
	}
}

// classifyErrorType determines the type of error
func classifyErrorType(err error) ErrorType {
	errStr := strings.ToLower(err.Error())

	// Network errors
	if isNetworkError(err) {
		return ErrorTypeNetwork
	}

	// Authentication/Authorization errors
	if isAuthError(errStr) {
		return ErrorTypeAuth
	}

	// Not found errors
	if isNotFoundError(errStr) {
		return ErrorTypeNotFound
	}

	// Validation errors
	if isValidationError(errStr) {
		return ErrorTypeValidation
	}

	// Timeout errors
	if isTimeoutError(err) {
		return ErrorTypeTimeout
	}

	return ErrorTypeUnknown
}

// isNetworkError checks if the error is network-related
func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no route to host",
		"network unreachable",
		"dial tcp",
		"i/o timeout",
		"broken pipe",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// isAuthError checks if the error is authentication-related
func isAuthError(errStr string) bool {
	authKeywords := []string{
		"unauthorized",
		"authentication failed",
		"invalid credentials",
		"access denied",
		"forbidden",
		"401",
		"403",
	}

	for _, keyword := range authKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// isNotFoundError checks if the error is a not found error
func isNotFoundError(errStr string) bool {
	notFoundKeywords := []string{
		"not found",
		"does not exist",
		"404",
		"no such",
	}

	for _, keyword := range notFoundKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// isValidationError checks if the error is a validation error
func isValidationError(errStr string) bool {
	validationKeywords := []string{
		"invalid",
		"validation failed",
		"bad request",
		"400",
		"malformed",
		"parse error",
	}

	for _, keyword := range validationKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// isTimeoutError checks if the error is a timeout error
func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded")
}

// isRetryable determines if an error should be retried
func isRetryable(errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout:
		return true
	case ErrorTypeAuth:
		return true // Retry auth errors (credentials might be updated)
	case ErrorTypeNotFound:
		return false // Resource not found errors are usually not transient
	case ErrorTypeValidation:
		return false // Validation errors won't fix themselves
	case ErrorTypeUnknown:
		return true // Retry unknown errors to be safe
	default:
		return true
	}
}

// getBackoffForErrorType returns the appropriate backoff for an error type
func getBackoffForErrorType(errorType ErrorType) wait.Backoff {
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout:
		return NetworkErrorBackoff
	case ErrorTypeAuth:
		return AuthErrorBackoff
	case ErrorTypeValidation:
		return ValidationErrorBackoff
	default:
		return DefaultBackoff
	}
}

// GetBackoffForError returns the backoff configuration for a given error
func GetBackoffForError(err error) wait.Backoff {
	retryErr := ClassifyError(err)
	if retryErr == nil {
		return DefaultBackoff
	}
	return getBackoffForErrorType(retryErr.Type)
}

// ShouldRetry determines if an operation should be retried based on the error
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	retryErr := ClassifyError(err)
	return retryErr.Retryable
}

// GetRetryDelay returns the delay before the next retry
func GetRetryDelay(err error, attemptNumber int) time.Duration {
	backoff := GetBackoffForError(err)

	// Calculate delay with exponential backoff
	delay := backoff.Duration
	for i := 1; i < attemptNumber && i < backoff.Steps; i++ {
		delay = time.Duration(float64(delay) * backoff.Factor)
		if delay > backoff.Cap {
			delay = backoff.Cap
			break
		}
	}

	// Add jitter
	if backoff.Jitter > 0 {
		jitterRange := float64(delay) * backoff.Jitter
		jitter := time.Duration(float64(-jitterRange) + (2 * jitterRange * float64(time.Now().UnixNano()%1000) / 1000.0))
		delay += jitter
	}

	return delay
}
