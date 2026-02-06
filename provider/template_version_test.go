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

func TestSendGridClient_CreateTemplateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		versionName    string
		subject        string
		htmlContent    string
		active         int
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create version",
			templateID:     "d-template-123",
			versionName:    "Version 1",
			subject:        "Hello {{name}}",
			htmlContent:    "<h1>Hello {{name}}</h1>",
			active:         1,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "version-id-123",
				"template_id": "d-template-123",
				"name": "Version 1",
				"subject": "Hello {{name}}",
				"html_content": "<h1>Hello {{name}}</h1>",
				"plain_content": "",
				"active": 1,
				"editor": "code",
				"generate_plain_content": false,
				"updated_at": "2026-02-04T12:00:00Z"
			}`,
			expectError: false,
		},
		{
			name:           "successful create inactive version",
			templateID:     "d-template-456",
			versionName:    "Draft Version",
			subject:        "Test Subject",
			htmlContent:    "<p>Test content</p>",
			active:         0,
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": "version-id-456",
				"template_id": "d-template-456",
				"name": "Draft Version",
				"subject": "Test Subject",
				"html_content": "<p>Test content</p>",
				"plain_content": "",
				"active": 0,
				"editor": "code",
				"generate_plain_content": false,
				"updated_at": "2026-02-04T12:00:00Z"
			}`,
			expectError: false,
		},
		{
			name:           "error - template not found",
			templateID:     "nonexistent-template",
			versionName:    "Version 1",
			subject:        "Test",
			htmlContent:    "<p>Test</p>",
			active:         1,
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError:   true,
			errorContains: "resource not found",
		},
		{
			name:           "error - unauthorized",
			templateID:     "d-template-123",
			versionName:    "Version 1",
			subject:        "Test",
			htmlContent:    "<p>Test</p>",
			active:         1,
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - bad request",
			templateID:     "d-template-123",
			versionName:    "",
			subject:        "",
			htmlContent:    "",
			active:         1,
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "name is required", "field": "name"}]
			}`,
			expectError:   true,
			errorContains: "name is required",
		},
		{
			name:           "error - rate limit exceeded",
			templateID:     "d-template-123",
			versionName:    "Version 1",
			subject:        "Test",
			htmlContent:    "<p>Test</p>",
			active:         1,
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
				assert.Equal(t, "/v3/templates/"+tt.templateID+"/versions", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.versionName, reqBody["name"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name":         tt.versionName,
				"subject":      tt.subject,
				"html_content": tt.htmlContent,
				"active":       tt.active,
			}

			var result struct {
				ID                   string `json:"id"`
				TemplateID           string `json:"template_id"`
				Name                 string `json:"name"`
				Subject              string `json:"subject"`
				HTMLContent          string `json:"html_content"`
				PlainContent         string `json:"plain_content"`
				Active               int    `json:"active"`
				Editor               string `json:"editor"`
				GeneratePlainContent bool   `json:"generate_plain_content"`
				UpdatedAt            string `json:"updated_at"`
			}

			err := client.Post(context.Background(), "/v3/templates/"+tt.templateID+"/versions", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.ID)
				assert.Equal(t, tt.templateID, result.TemplateID)
				assert.Equal(t, tt.versionName, result.Name)
			}
		})
	}
}

func TestSendGridClient_GetTemplateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		versionID      string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get version",
			templateID:     "d-template-123",
			versionID:      "version-id-123",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "version-id-123",
				"template_id": "d-template-123",
				"name": "Version 1",
				"subject": "Hello {{name}}",
				"html_content": "<h1>Hello {{name}}</h1>",
				"plain_content": "Hello {{name}}",
				"active": 1,
				"editor": "code",
				"generate_plain_content": true,
				"test_data": "{\"name\": \"Test\"}",
				"updated_at": "2026-02-04T12:00:00Z",
				"thumbnail_url": "https://example.com/thumb.png"
			}`,
			expectError: false,
		},
		{
			name:           "version not found",
			templateID:     "d-template-123",
			versionID:      "nonexistent-version",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError:    true,
			expectNotFound: true,
		},
		{
			name:           "template not found",
			templateID:     "nonexistent-template",
			versionID:      "version-id-123",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError:    true,
			expectNotFound: true,
		},
		{
			name:           "unauthorized",
			templateID:     "d-template-123",
			versionID:      "version-id-123",
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
				assert.Equal(t, "/v3/templates/"+tt.templateID+"/versions/"+tt.versionID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result struct {
				ID                   string `json:"id"`
				TemplateID           string `json:"template_id"`
				Name                 string `json:"name"`
				Subject              string `json:"subject"`
				HTMLContent          string `json:"html_content"`
				PlainContent         string `json:"plain_content"`
				Active               int    `json:"active"`
				Editor               string `json:"editor"`
				GeneratePlainContent bool   `json:"generate_plain_content"`
				TestData             string `json:"test_data"`
				UpdatedAt            string `json:"updated_at"`
				ThumbnailURL         string `json:"thumbnail_url"`
			}

			path := "/v3/templates/" + tt.templateID + "/versions/" + tt.versionID
			err := client.Get(context.Background(), path, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.versionID, result.ID)
				assert.Equal(t, tt.templateID, result.TemplateID)
			}
		})
	}
}

func TestSendGridClient_UpdateTemplateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		versionID      string
		newName        string
		newSubject     string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update name and subject",
			templateID:     "d-template-123",
			versionID:      "version-id-123",
			newName:        "Updated Version",
			newSubject:     "Updated Subject {{name}}",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "version-id-123",
				"template_id": "d-template-123",
				"name": "Updated Version",
				"subject": "Updated Subject {{name}}",
				"html_content": "<h1>Hello</h1>",
				"plain_content": "",
				"active": 1,
				"editor": "code",
				"generate_plain_content": false,
				"updated_at": "2026-02-04T13:00:00Z"
			}`,
			expectError: false,
		},
		{
			name:           "version not found",
			templateID:     "d-template-123",
			versionID:      "nonexistent-version",
			newName:        "Updated Name",
			newSubject:     "Updated Subject",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"errors": [{"message": "resource not found"}]
			}`,
			expectError: true,
		},
		{
			name:           "bad request - empty name",
			templateID:     "d-template-123",
			versionID:      "version-id-123",
			newName:        "",
			newSubject:     "Subject",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "name is required", "field": "name"}]
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/v3/templates/"+tt.templateID+"/versions/"+tt.versionID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.newName, reqBody["name"])
				assert.Equal(t, tt.newSubject, reqBody["subject"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name":    tt.newName,
				"subject": tt.newSubject,
			}

			var result struct {
				ID                   string `json:"id"`
				TemplateID           string `json:"template_id"`
				Name                 string `json:"name"`
				Subject              string `json:"subject"`
				HTMLContent          string `json:"html_content"`
				PlainContent         string `json:"plain_content"`
				Active               int    `json:"active"`
				Editor               string `json:"editor"`
				GeneratePlainContent bool   `json:"generate_plain_content"`
				UpdatedAt            string `json:"updated_at"`
			}

			path := "/v3/templates/" + tt.templateID + "/versions/" + tt.versionID
			err := client.Patch(context.Background(), path, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newName, result.Name)
				assert.Equal(t, tt.newSubject, result.Subject)
			}
		})
	}
}

func TestSendGridClient_DeleteTemplateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		templateID     string
		versionID      string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			templateID:     "d-template-123",
			versionID:      "version-id-123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "version not found (already deleted)",
			templateID:     "d-template-123",
			versionID:      "nonexistent-version",
			responseStatus: http.StatusNotFound,
			expectError:    true,
			expectNotFound: true,
		},
		{
			name:           "cannot delete last active version",
			templateID:     "d-template-123",
			versionID:      "version-id-123",
			responseStatus: http.StatusForbidden,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/v3/templates/"+tt.templateID+"/versions/"+tt.versionID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			path := "/v3/templates/" + tt.templateID + "/versions/" + tt.versionID
			err := client.Delete(context.Background(), path)

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

func TestTemplateVersionEditor_Constants(t *testing.T) {
	t.Parallel()

	// Test that the constants have expected values
	assert.Equal(t, TemplateVersionEditor("code"), TemplateVersionEditorCode)
	assert.Equal(t, TemplateVersionEditor("design"), TemplateVersionEditorDesign)
}

func TestBuildTemplateVersionState(t *testing.T) {
	t.Parallel()

	result := struct {
		ID                   string `json:"id"`
		TemplateID           string `json:"template_id"`
		Name                 string `json:"name"`
		Subject              string `json:"subject"`
		HTMLContent          string `json:"html_content"`
		PlainContent         string `json:"plain_content"`
		Active               int    `json:"active"`
		Editor               string `json:"editor"`
		GeneratePlainContent bool   `json:"generate_plain_content"`
		TestData             string `json:"test_data"`
		UpdatedAt            string `json:"updated_at"`
		ThumbnailURL         string `json:"thumbnail_url"`
	}{
		ID:                   "version-123",
		TemplateID:           "d-template-456",
		Name:                 "Test Version",
		Subject:              "Hello {{name}}",
		HTMLContent:          "<h1>Hello</h1>",
		PlainContent:         "Hello",
		Active:               1,
		Editor:               "code",
		GeneratePlainContent: true,
		TestData:             "{\"name\": \"Test\"}",
		UpdatedAt:            "2026-02-04T12:00:00Z",
		ThumbnailURL:         "https://example.com/thumb.png",
	}

	state := buildTemplateVersionState(result)

	assert.Equal(t, "version-123", state.VersionID)
	assert.Equal(t, "d-template-456", state.TemplateID)
	assert.Equal(t, "Test Version", state.Name)
	assert.NotNil(t, state.Subject)
	assert.Equal(t, "Hello {{name}}", *state.Subject)
	assert.NotNil(t, state.HTMLContent)
	assert.Equal(t, "<h1>Hello</h1>", *state.HTMLContent)
	assert.NotNil(t, state.PlainContent)
	assert.Equal(t, "Hello", *state.PlainContent)
	assert.NotNil(t, state.Active)
	assert.Equal(t, 1, *state.Active)
	assert.NotNil(t, state.Editor)
	assert.Equal(t, TemplateVersionEditorCode, *state.Editor)
	assert.NotNil(t, state.GeneratePlainContent)
	assert.True(t, *state.GeneratePlainContent)
	assert.NotNil(t, state.TestData)
	assert.Equal(t, "{\"name\": \"Test\"}", *state.TestData)
	assert.Equal(t, "2026-02-04T12:00:00Z", state.UpdatedAt)
	assert.Equal(t, "https://example.com/thumb.png", state.ThumbnailURL)
}

func TestBuildTemplateVersionState_EmptyFields(t *testing.T) {
	t.Parallel()

	result := struct {
		ID                   string `json:"id"`
		TemplateID           string `json:"template_id"`
		Name                 string `json:"name"`
		Subject              string `json:"subject"`
		HTMLContent          string `json:"html_content"`
		PlainContent         string `json:"plain_content"`
		Active               int    `json:"active"`
		Editor               string `json:"editor"`
		GeneratePlainContent bool   `json:"generate_plain_content"`
		TestData             string `json:"test_data"`
		UpdatedAt            string `json:"updated_at"`
		ThumbnailURL         string `json:"thumbnail_url"`
	}{
		ID:                   "version-123",
		TemplateID:           "d-template-456",
		Name:                 "Test Version",
		Subject:              "",
		HTMLContent:          "",
		PlainContent:         "",
		Active:               0,
		Editor:               "",
		GeneratePlainContent: false,
		TestData:             "",
		UpdatedAt:            "",
		ThumbnailURL:         "",
	}

	state := buildTemplateVersionState(result)

	assert.Equal(t, "version-123", state.VersionID)
	assert.Equal(t, "d-template-456", state.TemplateID)
	assert.Equal(t, "Test Version", state.Name)
	assert.Nil(t, state.Subject)
	assert.Nil(t, state.HTMLContent)
	assert.Nil(t, state.PlainContent)
	assert.NotNil(t, state.Active)
	assert.Equal(t, 0, *state.Active)
	assert.Nil(t, state.Editor)
	assert.NotNil(t, state.GeneratePlainContent)
	assert.False(t, *state.GeneratePlainContent)
	assert.Nil(t, state.TestData)
	assert.Equal(t, "", state.UpdatedAt)
	assert.Equal(t, "", state.ThumbnailURL)
}
