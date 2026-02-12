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
	"errors"
	"net"
	"testing"
	"time"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedType  ErrorType
		expectedRetry bool
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedType:  ErrorTypeUnknown,
			expectedRetry: false,
		},
		{
			name:          "network connection refused",
			err:           errors.New("connection refused"),
			expectedType:  ErrorTypeNetwork,
			expectedRetry: true,
		},
		{
			name:          "unauthorized error",
			err:           errors.New("401 unauthorized"),
			expectedType:  ErrorTypeAuth,
			expectedRetry: true,
		},
		{
			name:          "not found error",
			err:           errors.New("chart not found"),
			expectedType:  ErrorTypeNotFound,
			expectedRetry: false,
		},
		{
			name:          "validation error",
			err:           errors.New("invalid configuration"),
			expectedType:  ErrorTypeValidation,
			expectedRetry: false,
		},
		{
			name:          "forbidden error",
			err:           errors.New("403 forbidden"),
			expectedType:  ErrorTypeAuth,
			expectedRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)

			if tt.err == nil {
				if result != nil {
					t.Errorf("ClassifyError() for nil error = %v, want nil", result)
				}
				return
			}

			if result.Type != tt.expectedType {
				t.Errorf("ClassifyError().Type = %v, want %v", result.Type, tt.expectedType)
			}

			if result.Retryable != tt.expectedRetry {
				t.Errorf("ClassifyError().Retryable = %v, want %v", result.Retryable, tt.expectedRetry)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "dial tcp error",
			err:  errors.New("dial tcp: connection timeout"),
			want: true,
		},
		{
			name: "non-network error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "net.Error interface",
			err:  &net.DNSError{IsTimeout: true},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkError(tt.err); got != tt.want {
				t.Errorf("isNetworkError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRetryDelay(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		attemptNumber int
		wantMin       time.Duration
		wantMax       time.Duration
	}{
		{
			name:          "network error first attempt",
			err:           errors.New("connection refused"),
			attemptNumber: 1,
			wantMin:       4 * time.Second,
			wantMax:       6 * time.Second,
		},
		{
			name:          "network error third attempt",
			err:           errors.New("connection refused"),
			attemptNumber: 3,
			wantMin:       15 * time.Second,
			wantMax:       25 * time.Second,
		},
		{
			name:          "auth error first attempt",
			err:           errors.New("unauthorized"),
			attemptNumber: 1,
			wantMin:       50 * time.Second,
			wantMax:       70 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := GetRetryDelay(tt.err, tt.attemptNumber)

			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("GetRetryDelay() = %v, want between %v and %v", delay, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "network error",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "not found error",
			err:  errors.New("chart not found"),
			want: false,
		},
		{
			name: "validation error",
			err:  errors.New("invalid configuration"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRetry(tt.err); got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBackoffForError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedBackoff time.Duration
	}{
		{
			name:            "network error",
			err:             errors.New("connection refused"),
			expectedBackoff: NetworkErrorBackoff.Duration,
		},
		{
			name:            "auth error",
			err:             errors.New("unauthorized"),
			expectedBackoff: AuthErrorBackoff.Duration,
		},
		{
			name:            "validation error",
			err:             errors.New("invalid"),
			expectedBackoff: ValidationErrorBackoff.Duration,
		},
		{
			name:            "unknown error",
			err:             errors.New("something went wrong"),
			expectedBackoff: DefaultBackoff.Duration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := GetBackoffForError(tt.err)

			if backoff.Duration != tt.expectedBackoff {
				t.Errorf("GetBackoffForError().Duration = %v, want %v", backoff.Duration, tt.expectedBackoff)
			}
		})
	}
}
