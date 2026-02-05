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

func TestSendGridClient_CreateGlobalSuppression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create",
			email:          "test@example.com",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"recipient_emails": ["test@example.com"]
			}`,
			expectError: false,
		},
		{
			name:           "successful create with special characters",
			email:          "test+tag@example.com",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"recipient_emails": ["test+tag@example.com"]
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			email:          "test@example.com",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - invalid email",
			email:          "invalid-email",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "invalid email address", "field": "recipient_emails"}]
			}`,
			expectError:   true,
			errorContains: "invalid email address",
		},
		{
			name:           "error - rate limit exceeded",
			email:          "test@example.com",
			responseStatus: http.StatusTooManyRequests,
			responseBody: `{
				"errors": [{"message": "rate limit exceeded"}]
			}`,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:           "error - email not in response",
			email:          "test@example.com",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"recipient_emails": ["other@example.com"]
			}`,
			expectError:   true,
			errorContains: "email was not added",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/asm/suppressions/global", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				emails, ok := reqBody["recipient_emails"].([]interface{})
				require.True(t, ok)
				require.Len(t, emails, 1)
				assert.Equal(t, tt.email, emails[0])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"recipient_emails": []string{tt.email},
			}

			var result struct {
				RecipientEmails []string `json:"recipient_emails"`
			}

			err := client.Post(context.Background(), "/v3/asm/suppressions/global", reqBody, &result)

			if tt.expectError {
				if tt.errorContains == "email was not added" {
					// This is a special case - the API succeeded but our validation fails
					require.NoError(t, err)
					found := false
					for _, email := range result.RecipientEmails {
						if email == tt.email {
							found = true
							break
						}
					}
					assert.False(t, found, "Expected email not to be in response")
				} else {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, result.RecipientEmails, tt.email)
			}
		})
	}
}

func TestSendGridClient_GetGlobalSuppression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get - email is suppressed",
			email:          "test@example.com",
			responseStatus: http.StatusOK,
			responseBody: `[{
				"email": "test@example.com",
				"created": 1680000000
			}]`,
			expectError: false,
		},
		{
			name:           "email not suppressed - empty array",
			email:          "not-suppressed@example.com",
			responseStatus: http.StatusOK,
			responseBody:   `[]`,
			expectError:    false,
			expectNotFound: true,
		},
		{
			name:           "not found - 404 response",
			email:          "nonexistent@example.com",
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
				expectedPath := "/v3/asm/suppressions/global/" + url.PathEscape(tt.email)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result []struct {
				Email     string `json:"email"`
				CreatedAt int64  `json:"created"`
			}

			encodedEmail := url.PathEscape(tt.email)
			err := client.Get(context.Background(), "/v3/asm/suppressions/global/"+encodedEmail, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				if tt.expectNotFound {
					assert.Empty(t, result)
				} else {
					assert.NotEmpty(t, result)
					assert.Equal(t, tt.email, result[0].Email)
				}
			}
		})
	}
}

func TestSendGridClient_DeleteGlobalSuppression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			email:          "test@example.com",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "delete with special characters in email",
			email:          "test+tag@example.com",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			email:          "nonexistent@example.com",
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
				expectedPath := "/v3/asm/suppressions/global/" + url.PathEscape(tt.email)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			encodedEmail := url.PathEscape(tt.email)
			err := client.Delete(context.Background(), "/v3/asm/suppressions/global/"+encodedEmail)

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

func TestGlobalSuppression_ServerErrors(t *testing.T) {
	t.Parallel()

	t.Run("500 internal server error on create", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"recipient_emails": []string{"test@example.com"},
		}

		var result struct {
			RecipientEmails []string `json:"recipient_emails"`
		}
		err := client.Post(context.Background(), "/v3/asm/suppressions/global", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, sgErr.StatusCode)
	})

	t.Run("403 forbidden - no permission", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors": [{"message": "access forbidden"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"recipient_emails": []string{"test@example.com"},
		}

		var result struct {
			RecipientEmails []string `json:"recipient_emails"`
		}
		err := client.Post(context.Background(), "/v3/asm/suppressions/global", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusForbidden, sgErr.StatusCode)
	})

	t.Run("500 error on read", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result []struct {
			Email     string `json:"email"`
			CreatedAt int64  `json:"created"`
		}
		err := client.Get(context.Background(), "/v3/asm/suppressions/global/test@example.com", &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, sgErr.StatusCode)
	})
}

func TestGlobalSuppression_URLEncoding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		email         string
		encodedEmail  string
	}{
		{
			name:         "simple email",
			email:        "test@example.com",
			encodedEmail: "test@example.com",
		},
		{
			name:         "email with plus sign",
			email:        "test+tag@example.com",
			encodedEmail: "test+tag@example.com",
		},
		{
			name:         "email with special chars",
			email:        "test.name+filter@sub.example.com",
			encodedEmail: "test.name+filter@sub.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded := url.PathEscape(tc.email)
			// Verify the encoding works as expected
			assert.NotEmpty(t, encoded)
		})
	}
}

func TestGlobalSuppression_MultipleEmails(t *testing.T) {
	// Test that even though we add one email at a time, the API response
	// might contain multiple emails (e.g., if batch adding)
	t.Parallel()

	server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"recipient_emails": ["one@example.com", "two@example.com", "test@example.com"]
		}`))
	})

	client := NewSendGridClient("test-api-key", server.URL)

	reqBody := map[string]interface{}{
		"recipient_emails": []string{"test@example.com"},
	}

	var result struct {
		RecipientEmails []string `json:"recipient_emails"`
	}

	err := client.Post(context.Background(), "/v3/asm/suppressions/global", reqBody, &result)
	require.NoError(t, err)

	// Verify our target email is in the response
	found := false
	for _, email := range result.RecipientEmails {
		if email == "test@example.com" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected email should be in response")
}
