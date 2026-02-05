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

func TestSendGridClient_CreateEventWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		webhookURL     string
		enabled        *bool
		friendlyName   *string
		bounce         *bool
		delivered      *bool
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create with all events",
			webhookURL:     "https://example.com/webhook",
			enabled:        boolPtr(true),
			friendlyName:   strPtr("My Webhook"),
			bounce:         boolPtr(true),
			delivered:      boolPtr(true),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "webhook-123",
				"url": "https://example.com/webhook",
				"enabled": true,
				"friendly_name": "My Webhook",
				"bounce": true,
				"click": false,
				"deferred": false,
				"delivered": true,
				"dropped": false,
				"open": false,
				"processed": false,
				"spam_report": false,
				"unsubscribe": false,
				"group_resubscribe": false,
				"group_unsubscribe": false
			}`,
			expectError: false,
		},
		{
			name:           "successful create with minimal config",
			webhookURL:     "https://example.com/events",
			enabled:        nil,
			friendlyName:   nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "webhook-456",
				"url": "https://example.com/events",
				"enabled": true,
				"bounce": false,
				"click": false,
				"deferred": false,
				"delivered": false,
				"dropped": false,
				"open": false,
				"processed": false,
				"spam_report": false,
				"unsubscribe": false,
				"group_resubscribe": false,
				"group_unsubscribe": false
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			webhookURL:     "https://example.com/webhook",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - URL already in use",
			webhookURL:     "https://example.com/existing",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "URL is already in use by another webhook", "field": "url"}]
			}`,
			expectError:   true,
			errorContains: "URL is already in use",
		},
		{
			name:           "error - invalid URL",
			webhookURL:     "not-a-valid-url",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "invalid URL format", "field": "url"}]
			}`,
			expectError:   true,
			errorContains: "invalid URL",
		},
		{
			name:           "error - rate limit exceeded",
			webhookURL:     "https://example.com/webhook",
			responseStatus: http.StatusTooManyRequests,
			responseBody: `{
				"errors": [{"message": "rate limit exceeded"}]
			}`,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/user/webhooks/event/settings", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.webhookURL, reqBody["url"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			args := &EventWebhookArgs{
				URL:          tt.webhookURL,
				Enabled:      tt.enabled,
				FriendlyName: tt.friendlyName,
				Bounce:       tt.bounce,
				Delivered:    tt.delivered,
			}
			reqBody := args.buildRequestBody()

			var result eventWebhookAPIResponse
			err := client.Post(context.Background(), "/v3/user/webhooks/event/settings", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.ID)
				assert.Equal(t, tt.webhookURL, result.URL)
			}
		})
	}
}

func TestSendGridClient_GetEventWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		webhookID      string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get",
			webhookID:      "webhook-123",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "webhook-123",
				"url": "https://example.com/webhook",
				"enabled": true,
				"friendly_name": "My Webhook",
				"bounce": true,
				"click": true,
				"deferred": false,
				"delivered": true,
				"dropped": false,
				"open": true,
				"processed": false,
				"spam_report": true,
				"unsubscribe": true,
				"group_resubscribe": false,
				"group_unsubscribe": false
			}`,
			expectError: false,
		},
		{
			name:           "successful get disabled webhook",
			webhookID:      "webhook-456",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "webhook-456",
				"url": "https://example.com/disabled",
				"enabled": false,
				"bounce": false,
				"click": false,
				"deferred": false,
				"delivered": false,
				"dropped": false,
				"open": false,
				"processed": false,
				"spam_report": false,
				"unsubscribe": false,
				"group_resubscribe": false,
				"group_unsubscribe": false
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			webhookID:      "nonexistent",
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
				assert.Equal(t, "/v3/user/webhooks/event/settings/"+tt.webhookID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result eventWebhookAPIResponse
			err := client.Get(context.Background(), "/v3/user/webhooks/event/settings/"+tt.webhookID, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.webhookID, result.ID)
			}
		})
	}
}

func TestSendGridClient_UpdateEventWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		webhookID      string
		newURL         string
		newEnabled     bool
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update",
			webhookID:      "webhook-123",
			newURL:         "https://new-example.com/webhook",
			newEnabled:     true,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "webhook-123",
				"url": "https://new-example.com/webhook",
				"enabled": true,
				"friendly_name": "",
				"bounce": true,
				"click": false,
				"deferred": false,
				"delivered": true,
				"dropped": false,
				"open": false,
				"processed": false,
				"spam_report": false,
				"unsubscribe": false,
				"group_resubscribe": false,
				"group_unsubscribe": false
			}`,
			expectError: false,
		},
		{
			name:           "disable webhook",
			webhookID:      "webhook-123",
			newURL:         "https://example.com/webhook",
			newEnabled:     false,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "webhook-123",
				"url": "https://example.com/webhook",
				"enabled": false,
				"bounce": false,
				"click": false,
				"deferred": false,
				"delivered": false,
				"dropped": false,
				"open": false,
				"processed": false,
				"spam_report": false,
				"unsubscribe": false,
				"group_resubscribe": false,
				"group_unsubscribe": false
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			webhookID:      "nonexistent",
			newURL:         "https://example.com/webhook",
			newEnabled:     true,
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
				assert.Equal(t, "/v3/user/webhooks/event/settings/"+tt.webhookID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.newURL, reqBody["url"])
				assert.Equal(t, tt.newEnabled, reqBody["enabled"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			args := &EventWebhookArgs{
				URL:     tt.newURL,
				Enabled: &tt.newEnabled,
			}
			reqBody := args.buildRequestBody()

			var result eventWebhookAPIResponse
			err := client.Patch(context.Background(), "/v3/user/webhooks/event/settings/"+tt.webhookID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newURL, result.URL)
				assert.Equal(t, tt.newEnabled, result.Enabled)
			}
		})
	}
}

func TestSendGridClient_DeleteEventWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		webhookID      string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			webhookID:      "webhook-123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			webhookID:      "nonexistent",
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
				assert.Equal(t, "/v3/user/webhooks/event/settings/"+tt.webhookID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/user/webhooks/event/settings/"+tt.webhookID)

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

func TestEventWebhookAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("with all fields", func(t *testing.T) {
		t.Parallel()

		resp := eventWebhookAPIResponse{
			ID:               "webhook-123",
			URL:              "https://example.com/webhook",
			Enabled:          true,
			FriendlyName:     "Test Webhook",
			Bounce:           true,
			Click:            true,
			Deferred:         false,
			Delivered:        true,
			Dropped:          false,
			Open:             true,
			Processed:        false,
			SpamReport:       true,
			Unsubscribe:      true,
			GroupResubscribe: false,
			GroupUnsubscribe: true,
		}

		state := resp.toState()

		assert.Equal(t, "https://example.com/webhook", state.URL)
		assert.NotNil(t, state.Enabled)
		assert.True(t, *state.Enabled)
		assert.NotNil(t, state.FriendlyName)
		assert.Equal(t, "Test Webhook", *state.FriendlyName)
		assert.Equal(t, "webhook-123", state.WebhookID)
		assert.True(t, *state.Bounce)
		assert.True(t, *state.Click)
		assert.False(t, *state.Deferred)
		assert.True(t, *state.Delivered)
		assert.False(t, *state.Dropped)
		assert.True(t, *state.Open)
		assert.False(t, *state.Processed)
		assert.True(t, *state.SpamReport)
		assert.True(t, *state.Unsubscribe)
		assert.False(t, *state.GroupResubscribe)
		assert.True(t, *state.GroupUnsubscribe)
	})

	t.Run("with empty friendly name", func(t *testing.T) {
		t.Parallel()

		resp := eventWebhookAPIResponse{
			ID:           "webhook-456",
			URL:          "https://example.com/events",
			Enabled:      false,
			FriendlyName: "",
			Bounce:       false,
			Click:        false,
		}

		state := resp.toState()

		assert.Equal(t, "https://example.com/events", state.URL)
		assert.NotNil(t, state.Enabled)
		assert.False(t, *state.Enabled)
		assert.Nil(t, state.FriendlyName)
		assert.Equal(t, "webhook-456", state.WebhookID)
	})
}

func TestEventWebhookArgs_BuildRequestBody(t *testing.T) {
	t.Parallel()

	t.Run("full request body", func(t *testing.T) {
		t.Parallel()

		args := &EventWebhookArgs{
			URL:              "https://example.com/webhook",
			Enabled:          boolPtr(true),
			FriendlyName:     strPtr("My Webhook"),
			Bounce:           boolPtr(true),
			Click:            boolPtr(false),
			Deferred:         boolPtr(true),
			Delivered:        boolPtr(true),
			Dropped:          boolPtr(false),
			Open:             boolPtr(true),
			Processed:        boolPtr(false),
			SpamReport:       boolPtr(true),
			Unsubscribe:      boolPtr(true),
			GroupResubscribe: boolPtr(false),
			GroupUnsubscribe: boolPtr(true),
		}

		reqBody := args.buildRequestBody()

		assert.Equal(t, "https://example.com/webhook", reqBody["url"])
		assert.Equal(t, true, reqBody["enabled"])
		assert.Equal(t, "My Webhook", reqBody["friendly_name"])
		assert.Equal(t, true, reqBody["bounce"])
		assert.Equal(t, false, reqBody["click"])
		assert.Equal(t, true, reqBody["deferred"])
		assert.Equal(t, true, reqBody["delivered"])
		assert.Equal(t, false, reqBody["dropped"])
		assert.Equal(t, true, reqBody["open"])
		assert.Equal(t, false, reqBody["processed"])
		assert.Equal(t, true, reqBody["spam_report"])
		assert.Equal(t, true, reqBody["unsubscribe"])
		assert.Equal(t, false, reqBody["group_resubscribe"])
		assert.Equal(t, true, reqBody["group_unsubscribe"])
	})

	t.Run("minimal request body defaults enabled to true", func(t *testing.T) {
		t.Parallel()

		args := &EventWebhookArgs{
			URL: "https://example.com/webhook",
		}

		reqBody := args.buildRequestBody()

		assert.Equal(t, "https://example.com/webhook", reqBody["url"])
		assert.Equal(t, true, reqBody["enabled"])
		_, hasFriendlyName := reqBody["friendly_name"]
		assert.False(t, hasFriendlyName)
	})
}

func TestEventWebhook_ServerErrors(t *testing.T) {
	t.Parallel()

	t.Run("500 internal server error", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		args := &EventWebhookArgs{URL: "https://example.com/webhook"}
		reqBody := args.buildRequestBody()

		var result eventWebhookAPIResponse
		err := client.Post(context.Background(), "/v3/user/webhooks/event/settings", reqBody, &result)

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

		args := &EventWebhookArgs{URL: "https://example.com/webhook"}
		reqBody := args.buildRequestBody()

		var result eventWebhookAPIResponse
		err := client.Post(context.Background(), "/v3/user/webhooks/event/settings", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusForbidden, sgErr.StatusCode)
	})
}
