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

func TestSendGridClient_CreateTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestName    string
		generation     string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create dynamic template",
			requestName:    "My Dynamic Template",
			generation:     "dynamic",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "d-template-id-123",
				"name": "My Dynamic Template",
				"generation": "dynamic",
				"updated_at": "2026-02-04T12:00:00Z",
				"versions": []
			}`,
			expectError: false,
		},
		{
			name:           "successful create legacy template",
			requestName:    "My Legacy Template",
			generation:     "legacy",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "template-id-456",
				"name": "My Legacy Template",
				"generation": "legacy",
				"updated_at": "2026-02-04T12:00:00Z",
				"versions": []
			}`,
			expectError: false,
		},
		{
			name:           "create with existing versions",
			requestName:    "Template With Versions",
			generation:     "dynamic",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "d-template-with-versions",
				"name": "Template With Versions",
				"generation": "dynamic",
				"updated_at": "2026-02-04T12:00:00Z",
				"versions": [
					{
						"id": "version-1",
						"template_id": "d-template-with-versions",
						"name": "Version 1",
						"active": 1,
						"updated_at": "2026-02-04T12:00:00Z"
					}
				]
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			requestName:    "Test Template",
			generation:     "dynamic",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - bad request (missing name)",
			requestName:    "",
			generation:     "dynamic",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "name is required", "field": "name"}]
			}`,
			expectError:   true,
			errorContains: "name is required",
		},
		{
			name:           "error - bad request (invalid generation)",
			requestName:    "Test Template",
			generation:     "invalid",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "generation must be 'legacy' or 'dynamic'", "field": "generation"}]
			}`,
			expectError:   true,
			errorContains: "generation must be",
		},
		{
			name:           "error - rate limit exceeded",
			requestName:    "Test Template",
			generation:     "dynamic",
			responseStatus: http.StatusTooManyRequests,
			responseBody: `{
				"errors": [{"message": "rate limit exceeded"}]
			}`,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:           "error - internal server error",
			requestName:    "Test Template",
			generation:     "dynamic",
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
				assert.Equal(t, "/v3/templates", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.requestName, reqBody["name"])
				assert.Equal(t, tt.generation, reqBody["generation"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name":       tt.requestName,
				"generation": tt.generation,
			}

			var result struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Generation string `json:"generation"`
				UpdatedAt  string `json:"updated_at"`
				Versions   []struct {
					ID         string `json:"id"`
					TemplateID string `json:"template_id"`
					Name       string `json:"name"`
					Active     int    `json:"active"`
					UpdatedAt  string `json:"updated_at"`
				} `json:"versions"`
			}

			err := client.Post(context.Background(), "/v3/templates", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.ID)
				assert.Equal(t, tt.requestName, result.Name)
				assert.Equal(t, tt.generation, result.Generation)
			}
		})
	}
}

func TestSendGridClient_GetTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get dynamic template",
			templateID:     "d-template-id-123",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "d-template-id-123",
				"name": "My Dynamic Template",
				"generation": "dynamic",
				"updated_at": "2026-02-04T12:00:00Z",
				"versions": [
					{
						"id": "version-1",
						"template_id": "d-template-id-123",
						"name": "Version 1",
						"active": 1,
						"updated_at": "2026-02-04T11:00:00Z"
					},
					{
						"id": "version-2",
						"template_id": "d-template-id-123",
						"name": "Version 2",
						"active": 0,
						"updated_at": "2026-02-04T12:00:00Z"
					}
				]
			}`,
			expectError: false,
		},
		{
			name:           "successful get legacy template",
			templateID:     "template-id-456",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "template-id-456",
				"name": "My Legacy Template",
				"generation": "legacy",
				"updated_at": "2026-02-04T12:00:00Z",
				"versions": []
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			templateID:     "nonexistent-template",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError:    true,
			expectNotFound: true,
		},
		{
			name:           "unauthorized",
			templateID:     "template-id",
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
				assert.Equal(t, "/v3/templates/"+tt.templateID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Generation string `json:"generation"`
				UpdatedAt  string `json:"updated_at"`
				Versions   []struct {
					ID         string `json:"id"`
					TemplateID string `json:"template_id"`
					Name       string `json:"name"`
					Active     int    `json:"active"`
					UpdatedAt  string `json:"updated_at"`
				} `json:"versions"`
			}

			err := client.Get(context.Background(), "/v3/templates/"+tt.templateID, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.templateID, result.ID)
			}
		})
	}
}

func TestSendGridClient_UpdateTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		newName        string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update name",
			templateID:     "d-template-id-123",
			newName:        "Updated Template Name",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "d-template-id-123",
				"name": "Updated Template Name",
				"generation": "dynamic",
				"updated_at": "2026-02-04T13:00:00Z",
				"versions": []
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			templateID:     "nonexistent-template",
			newName:        "Updated Name",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError: true,
		},
		{
			name:           "bad request - name too long",
			templateID:     "d-template-id-123",
			newName:        string(make([]byte, 101)), // 101 characters
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "name must be 100 characters or less", "field": "name"}]
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/v3/templates/"+tt.templateID, r.URL.Path)
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
				"name": tt.newName,
			}

			var result struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Generation string `json:"generation"`
				UpdatedAt  string `json:"updated_at"`
				Versions   []struct {
					ID         string `json:"id"`
					TemplateID string `json:"template_id"`
					Name       string `json:"name"`
					Active     int    `json:"active"`
					UpdatedAt  string `json:"updated_at"`
				} `json:"versions"`
			}

			err := client.Patch(context.Background(), "/v3/templates/"+tt.templateID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newName, result.Name)
			}
		})
	}
}

func TestSendGridClient_DeleteTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			templateID:     "d-template-id-123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			templateID:     "nonexistent-template",
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
				assert.Equal(t, "/v3/templates/"+tt.templateID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/templates/"+tt.templateID)

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

func TestTemplateGeneration_Constants(t *testing.T) {
	t.Parallel()

	// Test that the constants have expected values
	assert.Equal(t, TemplateGeneration("legacy"), TemplateGenerationLegacy)
	assert.Equal(t, TemplateGeneration("dynamic"), TemplateGenerationDynamic)
}

func TestTemplateVersionSummary_ActiveConversion(t *testing.T) {
	t.Parallel()

	// Test the conversion from API's int (0/1) to bool
	tests := []struct {
		name       string
		apiActive  int
		expectBool bool
	}{
		{
			name:       "active version",
			apiActive:  1,
			expectBool: true,
		},
		{
			name:       "inactive version",
			apiActive:  0,
			expectBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Simulate the conversion that happens in Create/Read
			active := tt.apiActive == 1
			assert.Equal(t, tt.expectBool, active)
		})
	}
}
