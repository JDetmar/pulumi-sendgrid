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

func TestSendGridClient_CreateLinkBranding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domain         string
		subdomain      string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create with subdomain",
			domain:         "example.com",
			subdomain:      "email",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 12345,
				"user_id": 67890,
				"domain": "example.com",
				"subdomain": "email",
				"username": "testuser",
				"default": false,
				"valid": false,
				"legacy": false,
				"dns": {
					"owner_cname": {
						"valid": false,
						"type": "cname",
						"host": "email.example.com",
						"data": "sendgrid.net"
					},
					"brand_cname": {
						"valid": false,
						"type": "cname",
						"host": "12345.email.example.com",
						"data": "sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "successful create without subdomain",
			domain:         "example.org",
			subdomain:      "",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 12346,
				"user_id": 67890,
				"domain": "example.org",
				"subdomain": "",
				"username": "testuser",
				"default": true,
				"valid": false,
				"legacy": false,
				"dns": {
					"owner_cname": {
						"valid": false,
						"type": "cname",
						"host": "example.org",
						"data": "sendgrid.net"
					},
					"brand_cname": {
						"valid": false,
						"type": "cname",
						"host": "12346.example.org",
						"data": "sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			domain:         "test.com",
			subdomain:      "",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - domain already exists",
			domain:         "existing.com",
			subdomain:      "email",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "a]branded link already exists for this domain", "field": "domain"}]
			}`,
			expectError:   true,
			errorContains: "branded link already exists",
		},
		{
			name:           "error - rate limit exceeded",
			domain:         "test.com",
			subdomain:      "",
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
				assert.Equal(t, "/v3/whitelabel/links", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.domain, reqBody["domain"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"domain": tt.domain,
			}
			if tt.subdomain != "" {
				reqBody["subdomain"] = tt.subdomain
			}

			var result linkBrandingAPIResponse
			err := client.Post(context.Background(), "/v3/whitelabel/links", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result.ID)
				assert.Equal(t, tt.domain, result.Domain)
				assert.NotEmpty(t, result.DNS.OwnerCname.Host)
				assert.NotEmpty(t, result.DNS.BrandCname.Host)
			}
		})
	}
}

func TestSendGridClient_GetLinkBranding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		linkID         string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get",
			linkID:         "12345",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12345,
				"user_id": 67890,
				"domain": "example.com",
				"subdomain": "email",
				"username": "testuser",
				"default": false,
				"valid": true,
				"legacy": false,
				"dns": {
					"owner_cname": {
						"valid": true,
						"type": "cname",
						"host": "email.example.com",
						"data": "sendgrid.net"
					},
					"brand_cname": {
						"valid": true,
						"type": "cname",
						"host": "12345.email.example.com",
						"data": "sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			linkID:         "99999",
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
				assert.Equal(t, "/v3/whitelabel/links/"+tt.linkID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result linkBrandingAPIResponse
			err := client.Get(context.Background(), "/v3/whitelabel/links/"+tt.linkID, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, 12345, result.ID)
				assert.Equal(t, "example.com", result.Domain)
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSendGridClient_UpdateLinkBranding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		linkID         string
		makeDefault    bool
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update to default",
			linkID:         "12345",
			makeDefault:    true,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12345,
				"user_id": 67890,
				"domain": "example.com",
				"subdomain": "email",
				"username": "testuser",
				"default": true,
				"valid": true,
				"legacy": false,
				"dns": {
					"owner_cname": {
						"valid": true,
						"type": "cname",
						"host": "email.example.com",
						"data": "sendgrid.net"
					},
					"brand_cname": {
						"valid": true,
						"type": "cname",
						"host": "12345.email.example.com",
						"data": "sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			linkID:         "99999",
			makeDefault:    true,
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
				assert.Equal(t, "/v3/whitelabel/links/"+tt.linkID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.makeDefault, reqBody["default"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"default": tt.makeDefault,
			}

			var result linkBrandingAPIResponse
			err := client.Patch(context.Background(), "/v3/whitelabel/links/"+tt.linkID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, result.Default)
			}
		})
	}
}

func TestSendGridClient_DeleteLinkBranding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		linkID         string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			linkID:         "12345",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			linkID:         "99999",
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
				assert.Equal(t, "/v3/whitelabel/links/"+tt.linkID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/whitelabel/links/"+tt.linkID)

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

func TestLinkBrandingAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("with all fields populated", func(t *testing.T) {
		t.Parallel()

		resp := linkBrandingAPIResponse{
			ID:        12345,
			UserID:    67890,
			Domain:    "example.com",
			Subdomain: "email",
			Username:  "testuser",
			Default:   true,
			Valid:     true,
			Legacy:    false,
			DNS: linkBrandingDNSResponse{
				OwnerCname: linkBrandingDNSRecordResponse{
					Valid: true,
					Type:  "cname",
					Host:  "email.example.com",
					Data:  "sendgrid.net",
				},
				BrandCname: linkBrandingDNSRecordResponse{
					Valid: true,
					Type:  "cname",
					Host:  "12345.email.example.com",
					Data:  "sendgrid.net",
				},
			},
		}

		state := resp.toState()

		assert.Equal(t, 12345, state.LinkID)
		assert.Equal(t, 67890, state.UserID)
		assert.Equal(t, "example.com", state.Domain)
		assert.NotNil(t, state.Subdomain)
		assert.Equal(t, "email", *state.Subdomain)
		assert.Equal(t, "testuser", state.Username)
		assert.NotNil(t, state.Default)
		assert.True(t, *state.Default)
		assert.True(t, state.Valid)
		assert.False(t, state.Legacy)

		// Check DNS records
		assert.NotNil(t, state.OwnerCname)
		assert.True(t, state.OwnerCname.Valid)
		assert.Equal(t, "cname", state.OwnerCname.Type)
		assert.Equal(t, "email.example.com", state.OwnerCname.Host)

		assert.NotNil(t, state.BrandCname)
		assert.True(t, state.BrandCname.Valid)
		assert.Equal(t, "12345.email.example.com", state.BrandCname.Host)
	})

	t.Run("with minimal fields", func(t *testing.T) {
		t.Parallel()

		resp := linkBrandingAPIResponse{
			ID:       12346,
			UserID:   67890,
			Domain:   "minimal.com",
			Username: "testuser",
			Valid:    false,
			Legacy:   false,
			DNS: linkBrandingDNSResponse{
				OwnerCname: linkBrandingDNSRecordResponse{
					Valid: false,
					Type:  "cname",
					Host:  "minimal.com",
					Data:  "sendgrid.net",
				},
				BrandCname: linkBrandingDNSRecordResponse{
					Valid: false,
					Type:  "cname",
					Host:  "12346.minimal.com",
					Data:  "sendgrid.net",
				},
			},
		}

		state := resp.toState()

		assert.Equal(t, 12346, state.LinkID)
		assert.Equal(t, "minimal.com", state.Domain)
		assert.Nil(t, state.Subdomain)
		assert.Nil(t, state.Default)
		assert.False(t, state.Valid)

		// DNS records should still be present
		assert.NotNil(t, state.OwnerCname)
		assert.NotNil(t, state.BrandCname)
	})
}

func TestLinkBrandingValidation(t *testing.T) {
	t.Parallel()

	t.Run("validate link branding", func(t *testing.T) {
		t.Parallel()

		responseBody := `{
			"id": 12345,
			"valid": true,
			"validation_results": {
				"owner_cname": {
					"valid": true,
					"reason": null
				},
				"brand_cname": {
					"valid": true,
					"reason": null
				}
			}
		}`

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v3/whitelabel/links/12345/validate", r.URL.Path)
			assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(responseBody))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result struct {
			ID    int  `json:"id"`
			Valid bool `json:"valid"`
		}
		err := client.Post(context.Background(), "/v3/whitelabel/links/12345/validate", nil, &result)

		require.NoError(t, err)
		assert.Equal(t, 12345, result.ID)
		assert.True(t, result.Valid)
	})

	t.Run("validate link branding - failed validation", func(t *testing.T) {
		t.Parallel()

		responseBody := `{
			"id": 12345,
			"valid": false,
			"validation_results": {
				"owner_cname": {
					"valid": false,
					"reason": "Expected CNAME record not found"
				},
				"brand_cname": {
					"valid": true,
					"reason": null
				}
			}
		}`

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(responseBody))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result struct {
			ID    int  `json:"id"`
			Valid bool `json:"valid"`
		}
		err := client.Post(context.Background(), "/v3/whitelabel/links/12345/validate", nil, &result)

		require.NoError(t, err)
		assert.Equal(t, 12345, result.ID)
		assert.False(t, result.Valid)
	})
}
