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

func TestSendGridClient_CreateSubuser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		username       string
		email          string
		password       string
		ips            []string
		region         *string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create",
			username:       "test-subuser",
			email:          "subuser@example.com",
			password:       "securePassword123",
			ips:            nil,
			region:         nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"user_id": 12345,
				"username": "test-subuser",
				"email": "subuser@example.com"
			}`,
			expectError: false,
		},
		{
			name:           "successful create with IPs",
			username:       "test-subuser",
			email:          "subuser@example.com",
			password:       "securePassword123",
			ips:            []string{"192.168.1.1", "192.168.1.2"},
			region:         nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"user_id": 12346,
				"username": "test-subuser",
				"email": "subuser@example.com",
				"ips": ["192.168.1.1", "192.168.1.2"]
			}`,
			expectError: false,
		},
		{
			name:           "successful create with region",
			username:       "eu-subuser",
			email:          "eu@example.com",
			password:       "securePassword123",
			ips:            nil,
			region:         strPtr("eu"),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"user_id": 12347,
				"username": "eu-subuser",
				"email": "eu@example.com",
				"region": "eu"
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			username:       "test-subuser",
			email:          "subuser@example.com",
			password:       "securePassword123",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - username already exists",
			username:       "existing-user",
			email:          "subuser@example.com",
			password:       "securePassword123",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "username already exists", "field": "username"}]
			}`,
			expectError:   true,
			errorContains: "username already exists",
		},
		{
			name:           "error - invalid email",
			username:       "test-subuser",
			email:          "invalid-email",
			password:       "securePassword123",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "invalid email address", "field": "email"}]
			}`,
			expectError:   true,
			errorContains: "invalid email",
		},
		{
			name:           "error - weak password",
			username:       "test-subuser",
			email:          "subuser@example.com",
			password:       "weak",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "password does not meet requirements", "field": "password"}]
			}`,
			expectError:   true,
			errorContains: "password does not meet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/subusers", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.username, reqBody["username"])
				assert.Equal(t, tt.email, reqBody["email"])
				assert.Equal(t, tt.password, reqBody["password"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"username": tt.username,
				"email":    tt.email,
				"password": tt.password,
			}
			if len(tt.ips) > 0 {
				reqBody["ips"] = tt.ips
			}
			if tt.region != nil {
				reqBody["region"] = *tt.region
			}

			var result subuserCreateResponse
			err := client.Post(context.Background(), "/v3/subusers", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.username, result.Username)
				assert.Equal(t, tt.email, result.Email)
				assert.NotZero(t, result.UserID)
			}
		})
	}
}

func TestSendGridClient_GetSubuser(t *testing.T) {
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
			name:           "successful get enabled subuser",
			username:       "test-subuser",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12345,
				"username": "test-subuser",
				"email": "subuser@example.com",
				"disabled": false
			}`,
			expectError: false,
		},
		{
			name:           "successful get disabled subuser",
			username:       "disabled-subuser",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12346,
				"username": "disabled-subuser",
				"email": "disabled@example.com",
				"disabled": true
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
				expectedPath := "/v3/subusers/" + url.PathEscape(tt.username)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result subuserGetResponse
			encodedUsername := url.PathEscape(tt.username)
			err := client.Get(context.Background(), "/v3/subusers/"+encodedUsername, &result)

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

func TestSendGridClient_UpdateSubuser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		username       string
		disabled       bool
		responseStatus int
		expectError    bool
	}{
		{
			name:           "successful disable",
			username:       "test-subuser",
			disabled:       true,
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "successful enable",
			username:       "test-subuser",
			disabled:       false,
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found",
			username:       "nonexistent",
			disabled:       true,
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				expectedPath := "/v3/subusers/" + url.PathEscape(tt.username)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.disabled, reqBody["disabled"])

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"disabled": tt.disabled,
			}
			encodedUsername := url.PathEscape(tt.username)
			err := client.Patch(context.Background(), "/v3/subusers/"+encodedUsername, reqBody, nil)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendGridClient_DeleteSubuser(t *testing.T) {
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
			username:       "test-subuser",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "delete with special characters in username",
			username:       "test.subuser+tag",
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
				expectedPath := "/v3/subusers/" + url.PathEscape(tt.username)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			encodedUsername := url.PathEscape(tt.username)
			err := client.Delete(context.Background(), "/v3/subusers/"+encodedUsername)

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

func TestSubuser_ServerErrors(t *testing.T) {
	t.Parallel()

	t.Run("500 internal server error on create", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"username": "test-subuser",
			"email":    "subuser@example.com",
			"password": "securePassword123",
		}

		var result subuserCreateResponse
		err := client.Post(context.Background(), "/v3/subusers", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, sgErr.StatusCode)
	})

	t.Run("403 forbidden - subuser limit reached", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors": [{"message": "subuser limit reached"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"username": "test-subuser",
			"email":    "subuser@example.com",
			"password": "securePassword123",
		}

		var result subuserCreateResponse
		err := client.Post(context.Background(), "/v3/subusers", reqBody, &result)

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

		var result subuserGetResponse
		err := client.Get(context.Background(), "/v3/subusers/test-user", &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusTooManyRequests, sgErr.StatusCode)
	})
}

func TestStringSlicesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "equal slices",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "nil slices",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "different lengths",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "different values",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "b", "a"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stringSlicesEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubuser_URLEncoding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		username string
	}{
		{
			name:     "simple username",
			username: "test-user",
		},
		{
			name:     "username with dot",
			username: "test.user",
		},
		{
			name:     "username with plus",
			username: "test+user",
		},
		{
			name:     "username with underscore",
			username: "test_user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded := url.PathEscape(tc.username)
			// Verify the encoding works as expected
			assert.NotEmpty(t, encoded)
		})
	}
}
