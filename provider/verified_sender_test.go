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

func TestSendGridClient_CreateVerifiedSender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		nickname       string
		fromEmail      string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create with all fields",
			nickname:       "Test Sender",
			fromEmail:      "test@example.com",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 12345,
				"nickname": "Test Sender",
				"from_email": "test@example.com",
				"from_name": "Test Name",
				"reply_to": "reply@example.com",
				"reply_to_name": "Reply Name",
				"address": "123 Main St",
				"address2": "Suite 100",
				"city": "San Francisco",
				"state": "CA",
				"zip": "94105",
				"country": "USA",
				"verified": false,
				"locked": false
			}`,
			expectError: false,
		},
		{
			name:           "successful create with minimal fields",
			nickname:       "Minimal Sender",
			fromEmail:      "minimal@example.com",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 12346,
				"nickname": "Minimal Sender",
				"from_email": "minimal@example.com",
				"from_name": "",
				"reply_to": "reply@example.com",
				"reply_to_name": "",
				"address": "456 Oak Ave",
				"address2": "",
				"city": "New York",
				"state": "",
				"zip": "",
				"country": "USA",
				"verified": false,
				"locked": false
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			nickname:       "Test Sender",
			fromEmail:      "test@example.com",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - bad request (invalid email)",
			nickname:       "Test Sender",
			fromEmail:      "invalid-email",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "from_email must be a valid email", "field": "from_email"}]
			}`,
			expectError:   true,
			errorContains: "from_email must be a valid email",
		},
		{
			name:           "error - rate limit exceeded",
			nickname:       "Test Sender",
			fromEmail:      "test@example.com",
			responseStatus: http.StatusTooManyRequests,
			responseBody: `{
				"errors": [{"message": "rate limit exceeded"}]
			}`,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:           "error - internal server error",
			nickname:       "Test Sender",
			fromEmail:      "test@example.com",
			responseStatus: http.StatusInternalServerError,
			responseBody: `{
				"errors": [{"message": "internal server error"}]
			}`,
			expectError:   true,
			errorContains: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/verified_senders", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.nickname, reqBody["nickname"])
				assert.Equal(t, tt.fromEmail, reqBody["from_email"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"nickname":   tt.nickname,
				"from_email": tt.fromEmail,
				"reply_to":   "reply@example.com",
				"address":    "123 Main St",
				"city":       "San Francisco",
				"country":    "USA",
			}

			var result verifiedSenderAPIResponse
			err := client.Post(context.Background(), "/v3/verified_senders", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result.ID)
				assert.Equal(t, tt.nickname, result.Nickname)
				assert.Equal(t, tt.fromEmail, result.FromEmail)
				assert.False(t, result.Verified) // Should be false until verified via email
			}
		})
	}
}

func TestSendGridClient_ListVerifiedSenders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectError    bool
		expectCount    int
	}{
		{
			name:           "successful list with results",
			responseStatus: http.StatusOK,
			responseBody: `{
				"results": [
					{
						"id": 12345,
						"nickname": "Sender 1",
						"from_email": "sender1@example.com",
						"from_name": "Sender One",
						"reply_to": "reply1@example.com",
						"reply_to_name": "",
						"address": "123 Main St",
						"address2": "",
						"city": "San Francisco",
						"state": "CA",
						"zip": "94105",
						"country": "USA",
						"verified": true,
						"locked": false
					},
					{
						"id": 12346,
						"nickname": "Sender 2",
						"from_email": "sender2@example.com",
						"from_name": "",
						"reply_to": "reply2@example.com",
						"reply_to_name": "",
						"address": "456 Oak Ave",
						"address2": "",
						"city": "New York",
						"state": "NY",
						"zip": "10001",
						"country": "USA",
						"verified": false,
						"locked": false
					}
				]
			}`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:           "successful list with no results",
			responseStatus: http.StatusOK,
			responseBody: `{
				"results": []
			}`,
			expectError: false,
			expectCount: 0,
		},
		{
			name:           "unauthorized",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v3/verified_senders", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result struct {
				Results []verifiedSenderAPIResponse `json:"results"`
			}

			err := client.Get(context.Background(), "/v3/verified_senders", &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result.Results, tt.expectCount)
			}
		})
	}
}

func TestSendGridClient_UpdateVerifiedSender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		senderID       string
		newNickname    string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful update",
			senderID:       "12345",
			newNickname:    "Updated Sender",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12345,
				"nickname": "Updated Sender",
				"from_email": "test@example.com",
				"from_name": "Test Name",
				"reply_to": "reply@example.com",
				"reply_to_name": "Reply Name",
				"address": "123 Main St",
				"address2": "",
				"city": "San Francisco",
				"state": "CA",
				"zip": "94105",
				"country": "USA",
				"verified": true,
				"locked": false
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			senderID:       "99999",
			newNickname:    "Updated Sender",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError:    true,
			expectNotFound: true,
		},
		{
			name:           "bad request",
			senderID:       "12345",
			newNickname:    "",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "nickname is required", "field": "nickname"}]
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/v3/verified_senders/"+tt.senderID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.newNickname, reqBody["nickname"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"nickname":   tt.newNickname,
				"from_email": "test@example.com",
				"reply_to":   "reply@example.com",
				"address":    "123 Main St",
				"city":       "San Francisco",
				"country":    "USA",
			}

			var result verifiedSenderAPIResponse
			err := client.Patch(context.Background(), "/v3/verified_senders/"+tt.senderID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newNickname, result.Nickname)
			}
		})
	}
}

func TestSendGridClient_DeleteVerifiedSender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		senderID       string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			senderID:       "12345",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			senderID:       "99999",
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
				assert.Equal(t, "/v3/verified_senders/"+tt.senderID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/verified_senders/"+tt.senderID)

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

func TestVerifiedSenderAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("with all fields populated", func(t *testing.T) {
		t.Parallel()

		resp := verifiedSenderAPIResponse{
			ID:          12345,
			Nickname:    "Test Sender",
			FromEmail:   "test@example.com",
			FromName:    "Test Name",
			ReplyTo:     "reply@example.com",
			ReplyToName: "Reply Name",
			Address:     "123 Main St",
			Address2:    "Suite 100",
			City:        "San Francisco",
			State:       "CA",
			Zip:         "94105",
			Country:     "USA",
			Verified:    true,
			Locked:      false,
		}

		state := resp.toState()

		assert.Equal(t, 12345, state.SenderID)
		assert.Equal(t, "Test Sender", state.Nickname)
		assert.Equal(t, "test@example.com", state.FromEmail)
		assert.NotNil(t, state.FromName)
		assert.Equal(t, "Test Name", *state.FromName)
		assert.Equal(t, "reply@example.com", state.ReplyTo)
		assert.NotNil(t, state.ReplyToName)
		assert.Equal(t, "Reply Name", *state.ReplyToName)
		assert.Equal(t, "123 Main St", state.Address)
		assert.NotNil(t, state.Address2)
		assert.Equal(t, "Suite 100", *state.Address2)
		assert.Equal(t, "San Francisco", state.City)
		assert.NotNil(t, state.State)
		assert.Equal(t, "CA", *state.State)
		assert.NotNil(t, state.Zip)
		assert.Equal(t, "94105", *state.Zip)
		assert.Equal(t, "USA", state.Country)
		assert.True(t, state.Verified)
		assert.False(t, state.Locked)
	})

	t.Run("with minimal fields", func(t *testing.T) {
		t.Parallel()

		resp := verifiedSenderAPIResponse{
			ID:        12346,
			Nickname:  "Minimal Sender",
			FromEmail: "minimal@example.com",
			ReplyTo:   "reply@example.com",
			Address:   "456 Oak Ave",
			City:      "New York",
			Country:   "USA",
			Verified:  false,
			Locked:    false,
		}

		state := resp.toState()

		assert.Equal(t, 12346, state.SenderID)
		assert.Equal(t, "Minimal Sender", state.Nickname)
		assert.Equal(t, "minimal@example.com", state.FromEmail)
		assert.Nil(t, state.FromName)
		assert.Equal(t, "reply@example.com", state.ReplyTo)
		assert.Nil(t, state.ReplyToName)
		assert.Equal(t, "456 Oak Ave", state.Address)
		assert.Nil(t, state.Address2)
		assert.Equal(t, "New York", state.City)
		assert.Nil(t, state.State)
		assert.Nil(t, state.Zip)
		assert.Equal(t, "USA", state.Country)
		assert.False(t, state.Verified)
		assert.False(t, state.Locked)
	})
}
