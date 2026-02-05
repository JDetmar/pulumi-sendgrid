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

func TestSendGridClient_CreateUnsubscribeGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupName      string
		description    *string
		isDefault      *bool
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create with all fields",
			groupName:      "Marketing Emails",
			description:    strPtr("Promotional and marketing content"),
			isDefault:      boolPtr(false),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 123,
				"name": "Marketing Emails",
				"description": "Promotional and marketing content",
				"is_default": false,
				"unsubscribes": 0
			}`,
			expectError: false,
		},
		{
			name:           "successful create with name only",
			groupName:      "Newsletter",
			description:    nil,
			isDefault:      nil,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 456,
				"name": "Newsletter",
				"description": "",
				"is_default": false,
				"unsubscribes": 0
			}`,
			expectError: false,
		},
		{
			name:           "successful create as default",
			groupName:      "Default Group",
			description:    strPtr("The default unsubscribe group"),
			isDefault:      boolPtr(true),
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 789,
				"name": "Default Group",
				"description": "The default unsubscribe group",
				"is_default": true,
				"unsubscribes": 0
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			groupName:      "test-group",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - group already exists",
			groupName:      "existing-group",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "group name already exists", "field": "name"}]
			}`,
			expectError:   true,
			errorContains: "group name already exists",
		},
		{
			name:           "error - name too long",
			groupName:      "This is a very long group name that exceeds the limit",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "name must be no more than 30 characters", "field": "name"}]
			}`,
			expectError:   true,
			errorContains: "name must be no more than 30 characters",
		},
		{
			name:           "error - rate limit exceeded",
			groupName:      "test-group",
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
				assert.Equal(t, "/v3/asm/groups", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.groupName, reqBody["name"])
				if tt.description != nil {
					assert.Equal(t, *tt.description, reqBody["description"])
				}
				if tt.isDefault != nil {
					assert.Equal(t, *tt.isDefault, reqBody["is_default"])
				}

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name": tt.groupName,
			}
			if tt.description != nil {
				reqBody["description"] = *tt.description
			}
			if tt.isDefault != nil {
				reqBody["is_default"] = *tt.isDefault
			}

			var result unsubscribeGroupAPIResponse
			err := client.Post(context.Background(), "/v3/asm/groups", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.groupName, result.Name)
				assert.NotZero(t, result.ID)
			}
		})
	}
}

func TestSendGridClient_GetUnsubscribeGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupID        string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get",
			groupID:        "123",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 123,
				"name": "Marketing Emails",
				"description": "Promotional content",
				"is_default": false,
				"unsubscribes": 42
			}`,
			expectError: false,
		},
		{
			name:           "successful get default group",
			groupID:        "456",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 456,
				"name": "Default Group",
				"description": "",
				"is_default": true,
				"unsubscribes": 100
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			groupID:        "999",
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
				assert.Equal(t, "/v3/asm/groups/"+tt.groupID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result unsubscribeGroupAPIResponse
			err := client.Get(context.Background(), "/v3/asm/groups/"+tt.groupID, &result)

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
				assert.NotEmpty(t, result.Name)
			}
		})
	}
}

func TestSendGridClient_UpdateUnsubscribeGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupID        string
		newName        string
		newDescription *string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update name only",
			groupID:        "123",
			newName:        "Updated Name",
			newDescription: nil,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 123,
				"name": "Updated Name",
				"description": "Old description",
				"is_default": false,
				"unsubscribes": 42
			}`,
			expectError: false,
		},
		{
			name:           "successful update with description",
			groupID:        "456",
			newName:        "New Name",
			newDescription: strPtr("New description here"),
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 456,
				"name": "New Name",
				"description": "New description here",
				"is_default": true,
				"unsubscribes": 100
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			groupID:        "999",
			newName:        "New Name",
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
				assert.Equal(t, "/v3/asm/groups/"+tt.groupID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.newName, reqBody["name"])
				if tt.newDescription != nil {
					assert.Equal(t, *tt.newDescription, reqBody["description"])
				}

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name": tt.newName,
			}
			if tt.newDescription != nil {
				reqBody["description"] = *tt.newDescription
			}

			var result unsubscribeGroupAPIResponse
			err := client.Patch(context.Background(), "/v3/asm/groups/"+tt.groupID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newName, result.Name)
			}
		})
	}
}

func TestSendGridClient_DeleteUnsubscribeGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupID        string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			groupID:        "123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			groupID:        "999",
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
				assert.Equal(t, "/v3/asm/groups/"+tt.groupID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/asm/groups/"+tt.groupID)

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

func TestUnsubscribeGroupAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("with all fields", func(t *testing.T) {
		t.Parallel()

		resp := unsubscribeGroupAPIResponse{
			ID:           123,
			Name:         "Marketing",
			Description:  "Marketing emails",
			IsDefault:    false,
			Unsubscribes: 42,
		}

		state := resp.toState()

		assert.Equal(t, "Marketing", state.Name)
		assert.NotNil(t, state.Description)
		assert.Equal(t, "Marketing emails", *state.Description)
		assert.NotNil(t, state.IsDefault)
		assert.False(t, *state.IsDefault)
		assert.Equal(t, 123, state.GroupID)
		assert.Equal(t, 42, state.Unsubscribes)
	})

	t.Run("with empty description", func(t *testing.T) {
		t.Parallel()

		resp := unsubscribeGroupAPIResponse{
			ID:           456,
			Name:         "Newsletter",
			Description:  "",
			IsDefault:    true,
			Unsubscribes: 0,
		}

		state := resp.toState()

		assert.Equal(t, "Newsletter", state.Name)
		assert.Nil(t, state.Description)
		assert.NotNil(t, state.IsDefault)
		assert.True(t, *state.IsDefault)
		assert.Equal(t, 456, state.GroupID)
		assert.Equal(t, 0, state.Unsubscribes)
	})

	t.Run("as default group", func(t *testing.T) {
		t.Parallel()

		resp := unsubscribeGroupAPIResponse{
			ID:           789,
			Name:         "Default",
			Description:  "The default group",
			IsDefault:    true,
			Unsubscribes: 1000,
		}

		state := resp.toState()

		assert.True(t, *state.IsDefault)
		assert.Equal(t, 1000, state.Unsubscribes)
	})
}

func TestUnsubscribeGroup_ServerErrors(t *testing.T) {
	t.Parallel()

	t.Run("500 internal server error", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result unsubscribeGroupAPIResponse
		err := client.Get(context.Background(), "/v3/asm/groups/123", &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, sgErr.StatusCode)
	})

	t.Run("403 forbidden - no permission", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors": [{"message": "access forbidden"}]}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		reqBody := map[string]interface{}{
			"name": "test-group",
		}

		var result unsubscribeGroupAPIResponse
		err := client.Post(context.Background(), "/v3/asm/groups", reqBody, &result)

		require.Error(t, err)
		sgErr, ok := err.(*SendGridError)
		require.True(t, ok)
		assert.Equal(t, http.StatusForbidden, sgErr.StatusCode)
	})
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
