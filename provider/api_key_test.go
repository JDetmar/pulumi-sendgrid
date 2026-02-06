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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSendGridServer creates a test server that mocks SendGrid API responses
func mockSendGridServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func TestSendGridClient_CreateAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestName    string
		requestScopes  []string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create with scopes",
			requestName:    "My API Key",
			requestScopes:  []string{"mail.send", "alerts.read"},
			responseStatus: http.StatusCreated,
			responseBody: `{
				"api_key": "SG.xxxxxxxx.yyyyyyyy",
				"api_key_id": "test-key-id-123",
				"name": "My API Key",
				"scopes": ["mail.send", "alerts.read"]
			}`,
			expectError: false,
		},
		{
			name:           "successful create without scopes (full access)",
			requestName:    "Full Access Key",
			requestScopes:  nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"api_key": "SG.fullaccess.keyvalue",
				"api_key_id": "full-access-id",
				"name": "Full Access Key",
				"scopes": []
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			requestName:    "Test Key",
			requestScopes:  []string{"mail.send"},
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - forbidden (max keys reached)",
			requestName:    "Test Key",
			requestScopes:  []string{"mail.send"},
			responseStatus: http.StatusForbidden,
			responseBody: `{
				"errors": [{"message": "max API keys limit reached"}]
			}`,
			expectError:   true,
			errorContains: "max API keys limit reached",
		},
		{
			name:           "error - bad request",
			requestName:    "",
			requestScopes:  nil,
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "name is required", "field": "name"}]
			}`,
			expectError:   true,
			errorContains: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/api_keys", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.requestName, reqBody["name"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name": tt.requestName,
			}
			if len(tt.requestScopes) > 0 {
				reqBody["scopes"] = tt.requestScopes
			}

			var result struct {
				APIKey   string   `json:"api_key"`
				APIKeyID string   `json:"api_key_id"`
				Name     string   `json:"name"`
				Scopes   []string `json:"scopes"`
			}

			err := client.Post(context.Background(), "/v3/api_keys", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.APIKey)
				assert.NotEmpty(t, result.APIKeyID)
				assert.Equal(t, tt.requestName, result.Name)
			}
		})
	}
}

func TestSendGridClient_GetAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		apiKeyID       string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get",
			apiKeyID:       "test-key-id",
			responseStatus: http.StatusOK,
			responseBody: `{
				"api_key_id": "test-key-id",
				"name": "My API Key",
				"scopes": ["mail.send", "alerts.read"]
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			apiKeyID:       "nonexistent-key",
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
				assert.Equal(t, "/v3/api_keys/"+tt.apiKeyID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result struct {
				APIKeyID string   `json:"api_key_id"`
				Name     string   `json:"name"`
				Scopes   []string `json:"scopes"`
			}

			err := client.Get(context.Background(), "/v3/api_keys/"+tt.apiKeyID, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.apiKeyID, result.APIKeyID)
			}
		})
	}
}

func TestSendGridClient_UpdateAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		apiKeyID       string
		newName        string
		newScopes      []string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update name and scopes",
			apiKeyID:       "test-key-id",
			newName:        "Updated Key Name",
			newScopes:      []string{"mail.send"},
			responseStatus: http.StatusOK,
			responseBody: `{
				"api_key_id": "test-key-id",
				"name": "Updated Key Name",
				"scopes": ["mail.send"]
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			apiKeyID:       "nonexistent-key",
			newName:        "Updated Name",
			newScopes:      []string{"mail.send"},
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
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/v3/api_keys/"+tt.apiKeyID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.newName, reqBody["name"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name":   tt.newName,
				"scopes": tt.newScopes,
			}

			var result struct {
				APIKeyID string   `json:"api_key_id"`
				Name     string   `json:"name"`
				Scopes   []string `json:"scopes"`
			}

			err := client.Put(context.Background(), "/v3/api_keys/"+tt.apiKeyID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newName, result.Name)
			}
		})
	}
}

func TestSendGridClient_DeleteAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		apiKeyID       string
		responseStatus int
		expectError    bool
	}{
		{
			name:           "successful delete",
			apiKeyID:       "test-key-id",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			apiKeyID:       "nonexistent-key",
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/v3/api_keys/"+tt.apiKeyID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/api_keys/"+tt.apiKeyID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendGridError(t *testing.T) {
	t.Parallel()

	t.Run("error with details", func(t *testing.T) {
		err := &SendGridError{
			StatusCode: 400,
			Message:    "Bad Request",
			Errors: []SendGridErrorDetail{
				{Message: "name is required", Field: "name"},
			},
		}
		assert.Contains(t, err.Error(), "name is required")
		assert.Contains(t, err.Error(), "400")
		assert.False(t, err.IsNotFound())
	})

	t.Run("error without details", func(t *testing.T) {
		err := &SendGridError{
			StatusCode: 500,
			Message:    "Internal Server Error",
		}
		assert.Contains(t, err.Error(), "Internal Server Error")
		assert.Contains(t, err.Error(), "500")
		assert.False(t, err.IsNotFound())
	})

	t.Run("not found error", func(t *testing.T) {
		err := &SendGridError{
			StatusCode: 404,
			Message:    "Not Found",
		}
		assert.True(t, err.IsNotFound())
	})
}

func TestNewSendGridClient(t *testing.T) {
	t.Parallel()

	t.Run("with custom base URL", func(t *testing.T) {
		client := NewSendGridClient("api-key", "https://custom.example.com")
		assert.Equal(t, "https://custom.example.com", client.baseURL)
		assert.Equal(t, "api-key", client.apiKey)
	})

	t.Run("with empty base URL uses default", func(t *testing.T) {
		client := NewSendGridClient("api-key", "")
		assert.Equal(t, DefaultBaseURL, client.baseURL)
	})
}
