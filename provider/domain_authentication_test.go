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

func TestSendGridClient_CreateDomainAuthentication(t *testing.T) {
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
			name:           "successful create with automatic security",
			domain:         "example.com",
			subdomain:      "mail",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 12345,
				"user_id": 67890,
				"domain": "example.com",
				"subdomain": "mail",
				"username": "testuser",
				"ips": [],
				"custom_spf": false,
				"default": false,
				"automatic_security": true,
				"valid": false,
				"legacy": false,
				"dns": {
					"mail_cname": {
						"valid": false,
						"type": "cname",
						"host": "mail.example.com",
						"data": "u12345.wl.sendgrid.net"
					},
					"dkim1": {
						"valid": false,
						"type": "cname",
						"host": "s1._domainkey.example.com",
						"data": "s1.domainkey.u12345.wl.sendgrid.net"
					},
					"dkim2": {
						"valid": false,
						"type": "cname",
						"host": "s2._domainkey.example.com",
						"data": "s2.domainkey.u12345.wl.sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "successful create without automatic security",
			domain:         "example.org",
			subdomain:      "",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"id": 12346,
				"user_id": 67890,
				"domain": "example.org",
				"subdomain": "",
				"username": "testuser",
				"ips": ["192.168.1.1"],
				"custom_spf": true,
				"default": true,
				"automatic_security": false,
				"valid": false,
				"legacy": false,
				"dns": {
					"mail_cname": {
						"valid": false,
						"type": "mx",
						"host": "example.org",
						"data": "mx.sendgrid.net"
					},
					"dkim1": {
						"valid": false,
						"type": "txt",
						"host": "s1._domainkey.example.org",
						"data": "k=rsa; t=s; p=MIGfMA..."
					},
					"dkim2": {
						"valid": false,
						"type": "txt",
						"host": "s2._domainkey.example.org",
						"data": "k=rsa; t=s; p=MIGfMA..."
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
			subdomain:      "",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "domain has already been taken", "field": "domain"}]
			}`,
			expectError:   true,
			errorContains: "domain has already been taken",
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
				assert.Equal(t, "/v3/whitelabel/domains", r.URL.Path)
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

			var result domainAuthAPIResponse
			err := client.Post(context.Background(), "/v3/whitelabel/domains", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result.ID)
				assert.Equal(t, tt.domain, result.Domain)
				assert.NotEmpty(t, result.DNS.Dkim1.Host)
			}
		})
	}
}

func TestSendGridClient_GetDomainAuthentication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domainID       string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get",
			domainID:       "12345",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12345,
				"user_id": 67890,
				"domain": "example.com",
				"subdomain": "mail",
				"username": "testuser",
				"ips": [],
				"custom_spf": false,
				"default": false,
				"automatic_security": true,
				"valid": true,
				"legacy": false,
				"dns": {
					"mail_cname": {
						"valid": true,
						"type": "cname",
						"host": "mail.example.com",
						"data": "u12345.wl.sendgrid.net"
					},
					"dkim1": {
						"valid": true,
						"type": "cname",
						"host": "s1._domainkey.example.com",
						"data": "s1.domainkey.u12345.wl.sendgrid.net"
					},
					"dkim2": {
						"valid": true,
						"type": "cname",
						"host": "s2._domainkey.example.com",
						"data": "s2.domainkey.u12345.wl.sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			domainID:       "99999",
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
				assert.Equal(t, "/v3/whitelabel/domains/"+tt.domainID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result domainAuthAPIResponse
			err := client.Get(context.Background(), "/v3/whitelabel/domains/"+tt.domainID, &result)

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

func TestSendGridClient_UpdateDomainAuthentication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domainID       string
		makeDefault    bool
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful update to default",
			domainID:       "12345",
			makeDefault:    true,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": 12345,
				"user_id": 67890,
				"domain": "example.com",
				"subdomain": "mail",
				"username": "testuser",
				"ips": [],
				"custom_spf": false,
				"default": true,
				"automatic_security": true,
				"valid": true,
				"legacy": false,
				"dns": {
					"mail_cname": {
						"valid": true,
						"type": "cname",
						"host": "mail.example.com",
						"data": "u12345.wl.sendgrid.net"
					},
					"dkim1": {
						"valid": true,
						"type": "cname",
						"host": "s1._domainkey.example.com",
						"data": "s1.domainkey.u12345.wl.sendgrid.net"
					},
					"dkim2": {
						"valid": true,
						"type": "cname",
						"host": "s2._domainkey.example.com",
						"data": "s2.domainkey.u12345.wl.sendgrid.net"
					}
				}
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			domainID:       "99999",
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
				assert.Equal(t, "/v3/whitelabel/domains/"+tt.domainID, r.URL.Path)
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

			var result domainAuthAPIResponse
			err := client.Patch(context.Background(), "/v3/whitelabel/domains/"+tt.domainID, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, result.Default)
			}
		})
	}
}

func TestSendGridClient_DeleteDomainAuthentication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domainID       string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			domainID:       "12345",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			domainID:       "99999",
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
				assert.Equal(t, "/v3/whitelabel/domains/"+tt.domainID, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/whitelabel/domains/"+tt.domainID)

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

func TestDomainAuthAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("with all fields populated", func(t *testing.T) {
		t.Parallel()

		resp := domainAuthAPIResponse{
			ID:                12345,
			UserID:            67890,
			Domain:            "example.com",
			Subdomain:         "mail",
			Username:          "testuser",
			Ips:               []string{"192.168.1.1"},
			CustomSpf:         true,
			Default:           true,
			AutomaticSecurity: true,
			Valid:             true,
			Legacy:            false,
			DNS: domainAuthDNSResponse{
				MailCname: dnsRecordResponse{
					Valid: true,
					Type:  "cname",
					Host:  "mail.example.com",
					Data:  "u12345.wl.sendgrid.net",
				},
				Dkim1: dnsRecordResponse{
					Valid: true,
					Type:  "cname",
					Host:  "s1._domainkey.example.com",
					Data:  "s1.domainkey.u12345.wl.sendgrid.net",
				},
				Dkim2: dnsRecordResponse{
					Valid: true,
					Type:  "cname",
					Host:  "s2._domainkey.example.com",
					Data:  "s2.domainkey.u12345.wl.sendgrid.net",
				},
			},
		}

		state := resp.toState()

		assert.Equal(t, 12345, state.DomainID)
		assert.Equal(t, 67890, state.UserID)
		assert.Equal(t, "example.com", state.Domain)
		assert.NotNil(t, state.Subdomain)
		assert.Equal(t, "mail", *state.Subdomain)
		assert.Equal(t, "testuser", state.Username)
		assert.Equal(t, []string{"192.168.1.1"}, state.Ips)
		assert.NotNil(t, state.CustomSpf)
		assert.True(t, *state.CustomSpf)
		assert.NotNil(t, state.Default)
		assert.True(t, *state.Default)
		assert.NotNil(t, state.AutomaticSecurity)
		assert.True(t, *state.AutomaticSecurity)
		assert.True(t, state.Valid)
		assert.False(t, state.Legacy)

		// Check DNS records
		assert.NotNil(t, state.MailCname)
		assert.True(t, state.MailCname.Valid)
		assert.Equal(t, "cname", state.MailCname.Type)
		assert.Equal(t, "mail.example.com", state.MailCname.Host)

		assert.NotNil(t, state.Dkim1)
		assert.True(t, state.Dkim1.Valid)
		assert.Equal(t, "s1._domainkey.example.com", state.Dkim1.Host)

		assert.NotNil(t, state.Dkim2)
		assert.True(t, state.Dkim2.Valid)
		assert.Equal(t, "s2._domainkey.example.com", state.Dkim2.Host)
	})

	t.Run("with minimal fields", func(t *testing.T) {
		t.Parallel()

		resp := domainAuthAPIResponse{
			ID:       12346,
			UserID:   67890,
			Domain:   "minimal.com",
			Username: "testuser",
			Valid:    false,
			Legacy:   false,
			DNS: domainAuthDNSResponse{
				MailCname: dnsRecordResponse{
					Valid: false,
					Type:  "cname",
					Host:  "mail.minimal.com",
					Data:  "u12346.wl.sendgrid.net",
				},
				Dkim1: dnsRecordResponse{
					Valid: false,
					Type:  "cname",
					Host:  "s1._domainkey.minimal.com",
					Data:  "s1.domainkey.u12346.wl.sendgrid.net",
				},
				Dkim2: dnsRecordResponse{
					Valid: false,
					Type:  "cname",
					Host:  "s2._domainkey.minimal.com",
					Data:  "s2.domainkey.u12346.wl.sendgrid.net",
				},
			},
		}

		state := resp.toState()

		assert.Equal(t, 12346, state.DomainID)
		assert.Equal(t, "minimal.com", state.Domain)
		assert.Nil(t, state.Subdomain)
		assert.Nil(t, state.CustomSpf)
		assert.Nil(t, state.Default)
		assert.Nil(t, state.AutomaticSecurity)
		assert.False(t, state.Valid)

		// DNS records should still be present
		assert.NotNil(t, state.MailCname)
		assert.NotNil(t, state.Dkim1)
		assert.NotNil(t, state.Dkim2)
	})
}
