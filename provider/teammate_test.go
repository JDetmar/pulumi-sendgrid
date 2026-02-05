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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendGridClient_InviteTeammate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		scopes         []string
		isAdmin        *bool
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful invite with scopes",
			email:          "teammate@example.com",
			scopes:         []string{"mail.send", "alerts.read"},
			isAdmin:        boolPtr(false),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"email": "teammate@example.com",
				"scopes": ["mail.send", "alerts.read"],
				"is_admin": false,
				"token": "invitation-token-123"
			}`,
			expectError: false,
		},
		{
			name:           "successful invite as admin",
			email:          "admin@example.com",
			scopes:         nil,
			isAdmin:        boolPtr(true),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"email": "admin@example.com",
				"scopes": [],
				"is_admin": true,
				"token": "admin-token-456"
			}`,
			expectError: false,
		},
		{
			name:           "successful invite minimal",
			email:          "minimal@example.com",
			scopes:         nil,
			isAdmin:        nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"email": "minimal@example.com",
				"scopes": [],
				"is_admin": false,
				"token": "minimal-token-789"
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			email:          "teammate@example.com",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - teammate already exists",
			email:          "existing@example.com",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "teammate already exists", "field": "email"}]
			}`,
			expectError:   true,
			errorContains: "teammate already exists",
		},
		{
			name:           "error - teammate limit reached",
			email:          "teammate@example.com",
			responseStatus: http.StatusForbidden,
			responseBody: `{
				"errors": [{"message": "teammate limit reached for your plan"}]
			}`,
			expectError:   true,
			errorContains: "teammate limit reached",
		},
		{
			name:           "error - invalid email",
			email:          "invalid-email",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "invalid email address", "field": "email"}]
			}`,
			expectError:   true,
			errorContains: "invalid email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/teammates", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.email, reqBody["email"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"email": tt.email,
			}
			if len(tt.scopes) > 0 {
				reqBody["scopes"] = tt.scopes
			}
			if tt.isAdmin != nil {
				reqBody["is_admin"] = *tt.isAdmin
			}

			var result teammateInviteResponse
			err := client.Post(context.Background(), "/v3/teammates", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.email, result.Email)
				assert.NotEmpty(t, result.Token)
			}
		})
	}
}

func TestSendGridClient_GetTeammate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		username       string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get teammate",
			username:       "teammate-user",
			responseStatus: http.StatusOK,
			responseBody: `{
				"username": "teammate-user",
				"email": "teammate@example.com",
				"first_name": "John",
				"last_name": "Doe",
				"scopes": ["mail.send", "alerts.read"],
				"user_type": "teammate",
				"is_admin": false
			}`,
			expectError: false,
		},
		{
			name:           "successful get admin teammate",
			username:       "admin-user",
			responseStatus: http.StatusOK,
			responseBody: `{
				"username": "admin-user",
				"email": "admin@example.com",
				"first_name": "Admin",
				"last_name": "User",
				"scopes": [],
				"user_type": "admin",
				"is_admin": true
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			username:       "nonexistent",
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
				expectedPath := "/v3/teammates/" + url.PathEscape(tt.username)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result teammateGetResponse
			encodedUsername := url.PathEscape(tt.username)
			err := client.Get(context.Background(), "/v3/teammates/"+encodedUsername, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.username, result.Username)
			}
		})
	}
}

func TestSendGridClient_UpdateTeammate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		username       string
		scopes         []string
		isAdmin        bool
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update scopes",
			username:       "teammate-user",
			scopes:         []string{"mail.send", "templates.read"},
			isAdmin:        false,
			responseStatus: http.StatusOK,
			responseBody: `{
				"username": "teammate-user",
				"email": "teammate@example.com",
				"first_name": "John",
				"last_name": "Doe",
				"scopes": ["mail.send", "templates.read"],
				"user_type": "teammate",
				"is_admin": false
			}`,
			expectError: false,
		},
		{
			name:           "promote to admin",
			username:       "teammate-user",
			scopes:         nil,
			isAdmin:        true,
			responseStatus: http.StatusOK,
			responseBody: `{
				"username": "teammate-user",
				"email": "teammate@example.com",
				"first_name": "John",
				"last_name": "Doe",
				"scopes": [],
				"user_type": "admin",
				"is_admin": true
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			username:       "nonexistent",
			scopes:         []string{"mail.send"},
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
				expectedPath := "/v3/teammates/" + url.PathEscape(tt.username)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"is_admin": tt.isAdmin,
			}
			if len(tt.scopes) > 0 {
				reqBody["scopes"] = tt.scopes
			}

			encodedUsername := url.PathEscape(tt.username)
			var result teammateGetResponse
			err := client.Patch(context.Background(), "/v3/teammates/"+encodedUsername, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.username, result.Username)
			}
		})
	}
}

func TestSendGridClient_DeleteTeammate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		username       string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			username:       "teammate-user",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			username:       "nonexistent",
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
				expectedPath := "/v3/teammates/" + url.PathEscape(tt.username)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			encodedUsername := url.PathEscape(tt.username)
			err := client.Delete(context.Background(), "/v3/teammates/"+encodedUsername)

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

func TestSendGridClient_DeletePendingInvitation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		token          string
		responseStatus int
		expectError    bool
	}{
		{
			name:           "successful delete pending invitation",
			token:          "invitation-token-123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already accepted or expired)",
			token:          "expired-token",
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/teammates/pending/" + tt.token
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/teammates/pending/"+tt.token)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendGridClient_GetPendingTeammates(t *testing.T) {
	t.Parallel()

	server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v3/teammates/pending", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"email": "pending1@example.com",
				"scopes": ["mail.send"],
				"is_admin": false,
				"token": "token-1"
			},
			{
				"email": "pending2@example.com",
				"scopes": [],
				"is_admin": true,
				"token": "token-2"
			}
		]`))
	})

	client := NewSendGridClient("test-api-key", server.URL)

	var result []struct {
		Email   string   `json:"email"`
		Scopes  []string `json:"scopes,omitempty"`
		IsAdmin bool     `json:"is_admin"`
		Token   string   `json:"token"`
	}
	err := client.Get(context.Background(), "/v3/teammates/pending", &result)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "pending1@example.com", result[0].Email)
	assert.Equal(t, "pending2@example.com", result[1].Email)
}

func TestTeammate_ServerErrors(t *testing.T) {
	t.Parallel()

	t.Run("500 internal server error", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"email": "teammate@example.com",
		}

		var result teammateInviteResponse
		err := client.Post(context.Background(), "/v3/teammates", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, sgErr.StatusCode)
	})

	t.Run("429 rate limit exceeded", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors": [{"message": "rate limit exceeded"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result []teammateGetResponse
		err := client.Get(context.Background(), "/v3/teammates", &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusTooManyRequests, sgErr.StatusCode)
	})
}
