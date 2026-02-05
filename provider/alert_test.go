// Copyright 2025, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendGridClient_CreateAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		alertType      string
		emailTo        string
		percentage     *int
		frequency      *string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create usage_limit alert",
			alertType:      "usage_limit",
			emailTo:        "alerts@example.com",
			percentage:     intPtr(90),
			frequency:      nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 123,
				"type": "usage_limit",
				"email_to": "alerts@example.com",
				"percentage": 90,
				"created_at": 1680000000,
				"updated_at": 1680000000
			}`,
			expectError: false,
		},
		{
			name:           "successful create stats_notification alert - daily",
			alertType:      "stats_notification",
			emailTo:        "stats@example.com",
			percentage:     nil,
			frequency:      strPtr("daily"),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 456,
				"type": "stats_notification",
				"email_to": "stats@example.com",
				"frequency": "daily",
				"created_at": 1680000000,
				"updated_at": 1680000000
			}`,
			expectError: false,
		},
		{
			name:           "successful create stats_notification alert - weekly",
			alertType:      "stats_notification",
			emailTo:        "stats@example.com",
			percentage:     nil,
			frequency:      strPtr("weekly"),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 789,
				"type": "stats_notification",
				"email_to": "stats@example.com",
				"frequency": "weekly",
				"created_at": 1680000000,
				"updated_at": 1680000000
			}`,
			expectError: false,
		},
		{
			name:           "successful create stats_notification alert - monthly",
			alertType:      "stats_notification",
			emailTo:        "stats@example.com",
			percentage:     nil,
			frequency:      strPtr("monthly"),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 101,
				"type": "stats_notification",
				"email_to": "stats@example.com",
				"frequency": "monthly",
				"created_at": 1680000000,
				"updated_at": 1680000000
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			alertType:      "usage_limit",
			emailTo:        "alerts@example.com",
			percentage:     intPtr(90),
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - invalid alert type",
			alertType:      "invalid_type",
			emailTo:        "alerts@example.com",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "invalid alert type", "field": "type"}]
			}`,
			expectError:   true,
			errorContains: "invalid alert type",
		},
		{
			name:           "error - invalid email",
			alertType:      "usage_limit",
			emailTo:        "invalid-email",
			percentage:     intPtr(90),
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "invalid email address", "field": "email_to"}]
			}`,
			expectError:   true,
			errorContains: "invalid email",
		},
		{
			name:           "error - invalid percentage",
			alertType:      "usage_limit",
			emailTo:        "alerts@example.com",
			percentage:     intPtr(150),
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "percentage must be between 1 and 100", "field": "percentage"}]
			}`,
			expectError:   true,
			errorContains: "percentage must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/alerts", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.alertType, reqBody["type"])
				assert.Equal(t, tt.emailTo, reqBody["email_to"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"type":     tt.alertType,
				"email_to": tt.emailTo,
			}
			if tt.percentage != nil {
				reqBody["percentage"] = *tt.percentage
			}
			if tt.frequency != nil {
				reqBody["frequency"] = *tt.frequency
			}

			var result alertAPIResponse
			err := client.Post(context.Background(), "/v3/alerts", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result.ID)
				assert.Equal(t, tt.alertType, result.Type)
				assert.Equal(t, tt.emailTo, result.EmailTo)
			}
		})
	}
}

func TestSendGridClient_GetAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		alertID        string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get usage_limit alert",
			alertID:        "123",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 123,
				"type": "usage_limit",
				"email_to": "alerts@example.com",
				"percentage": 90,
				"created_at": 1680000000,
				"updated_at": 1680000000
			}`,
			expectError: false,
		},
		{
			name:           "successful get stats_notification alert",
			alertID:        "456",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 456,
				"type": "stats_notification",
				"email_to": "stats@example.com",
				"frequency": "daily",
				"created_at": 1680000000,
				"updated_at": 1680000000
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			alertID:        "999",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError:    true,
			expectNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v3/alerts/"+tt.alertID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result alertAPIResponse
			err := client.Get(context.Background(), "/v3/alerts/"+tt.alertID, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result.ID)
			}
		})
	}
}

func TestSendGridClient_UpdateAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		alertID        string
		emailTo        string
		percentage     *int
		frequency      *string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update usage_limit alert",
			alertID:        "123",
			emailTo:        "new-alerts@example.com",
			percentage:     intPtr(95),
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 123,
				"type": "usage_limit",
				"email_to": "new-alerts@example.com",
				"percentage": 95,
				"created_at": 1680000000,
				"updated_at": 1680001000
			}`,
			expectError: false,
		},
		{
			name:           "successful update stats_notification frequency",
			alertID:        "456",
			emailTo:        "stats@example.com",
			frequency:      strPtr("weekly"),
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 456,
				"type": "stats_notification",
				"email_to": "stats@example.com",
				"frequency": "weekly",
				"created_at": 1680000000,
				"updated_at": 1680001000
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			alertID:        "999",
			emailTo:        "new@example.com",
			percentage:     intPtr(50),
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/v3/alerts/"+tt.alertID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.emailTo, reqBody["email_to"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"email_to": tt.emailTo,
			}
			if tt.percentage != nil {
				reqBody["percentage"] = *tt.percentage
			}
			if tt.frequency != nil {
				reqBody["frequency"] = *tt.frequency
			}

			var result alertAPIResponse
			err := client.Patch(context.Background(), "/v3/alerts/"+tt.alertID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.emailTo, result.EmailTo)
			}
		})
	}
}

func TestSendGridClient_DeleteAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		alertID        string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			alertID:        "123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			alertID:        "999",
			responseStatus: http.StatusNotFound,
			expectError:    true,
			expectNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/v3/alerts/"+tt.alertID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/alerts/"+tt.alertID)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("usage_limit alert", func(t *testing.T) {
		t.Parallel()

		resp := alertAPIResponse{
			ID:         123,
			Type:       "usage_limit",
			EmailTo:    "alerts@example.com",
			Percentage: 90,
			CreatedAt:  1680000000,
			UpdatedAt:  1680001000,
		}

		state := resp.toState()

		assert.Equal(t, "usage_limit", state.Type)
		assert.Equal(t, "alerts@example.com", state.EmailTo)
		assert.NotNil(t, state.Percentage)
		assert.Equal(t, 90, *state.Percentage)
		assert.Nil(t, state.Frequency)
		assert.Equal(t, 123, state.AlertID)
		assert.Equal(t, int64(1680000000), state.CreatedAt)
		assert.Equal(t, int64(1680001000), state.UpdatedAt)
	})

	t.Run("stats_notification alert", func(t *testing.T) {
		t.Parallel()

		resp := alertAPIResponse{
			ID:        456,
			Type:      "stats_notification",
			EmailTo:   "stats@example.com",
			Frequency: "daily",
			CreatedAt: 1680000000,
			UpdatedAt: 1680001000,
		}

		state := resp.toState()

		assert.Equal(t, "stats_notification", state.Type)
		assert.Equal(t, "stats@example.com", state.EmailTo)
		assert.Nil(t, state.Percentage)
		assert.NotNil(t, state.Frequency)
		assert.Equal(t, "daily", *state.Frequency)
		assert.Equal(t, 456, state.AlertID)
	})
}

func TestAlert_ServerErrors(t *testing.T) {
	t.Parallel()

	t.Run("500 internal server error", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"type":       "usage_limit",
			"email_to":   "alerts@example.com",
			"percentage": 90,
		}

		var result alertAPIResponse
		err := client.Post(context.Background(), "/v3/alerts", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, sgErr.StatusCode)
	})

	t.Run("403 forbidden", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors": [{"message": "access forbidden"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"type":       "usage_limit",
			"email_to":   "alerts@example.com",
			"percentage": 90,
		}

		var result alertAPIResponse
		err := client.Post(context.Background(), "/v3/alerts", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusForbidden, sgErr.StatusCode)
	})

	t.Run("429 rate limit exceeded", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors": [{"message": "rate limit exceeded"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result []alertAPIResponse
		err := client.Get(context.Background(), "/v3/alerts", &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusTooManyRequests, sgErr.StatusCode)
	})
}

// Helper function for int pointers
func intPtr(i int) *int {
	return &i
}
