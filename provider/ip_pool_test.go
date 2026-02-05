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

func TestSendGridClient_CreateIpPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		poolName       string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful create",
			poolName:       "marketing",
			responseStatus: http.StatusOK,
			responseBody: `{
				"pool_name": "marketing",
				"ips": []
			}`,
			expectError: false,
		},
		{
			name:           "successful create with IPs",
			poolName:       "transactional",
			responseStatus: http.StatusOK,
			responseBody: `{
				"pool_name": "transactional",
				"ips": ["167.89.21.3", "167.89.22.4"]
			}`,
			expectError: false,
		},
		{
			name:           "error - unauthorized",
			poolName:       "test-pool",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"errors": [{"message": "authorization required"}]
			}`,
			expectError:   true,
			errorContains: "authorization required",
		},
		{
			name:           "error - pool already exists",
			poolName:       "existing-pool",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"errors": [{"message": "pool already exists", "field": "name"}]
			}`,
			expectError:   true,
			errorContains: "pool already exists",
		},
		{
			name:           "error - rate limit exceeded",
			poolName:       "test-pool",
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
				assert.Equal(t, "/v3/ips/pools", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse and verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.poolName, reqBody["name"])

				// Send response
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name": tt.poolName,
			}

			var result ipPoolAPIResponse
			err := client.Post(context.Background(), "/v3/ips/pools", reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.poolName, result.PoolName)
			}
		})
	}
}

func TestSendGridClient_GetIpPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		poolName       string
		responseStatus int
		responseBody   string
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful get",
			poolName:       "marketing",
			responseStatus: http.StatusOK,
			responseBody: `{
				"pool_name": "marketing",
				"ips": ["167.89.21.3"]
			}`,
			expectError: false,
		},
		{
			name:           "successful get empty pool",
			poolName:       "new-pool",
			responseStatus: http.StatusOK,
			responseBody: `{
				"pool_name": "new-pool",
				"ips": []
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			poolName:       "nonexistent",
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
				assert.Equal(t, "/v3/ips/pools/"+tt.poolName, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			var result ipPoolAPIResponse
			err := client.Get(context.Background(), "/v3/ips/pools/"+tt.poolName, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectNotFound {
					sgErr, ok := err.(*SendGridError)
					require.True(t, ok)
					assert.True(t, sgErr.IsNotFound())
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.poolName, result.PoolName)
			}
		})
	}
}

func TestSendGridClient_UpdateIpPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		oldPoolName    string
		newPoolName    string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful rename",
			oldPoolName:    "old-name",
			newPoolName:    "new-name",
			responseStatus: http.StatusOK,
			responseBody: `{
				"name": "new-name"
			}`,
			expectError: false,
		},
		{
			name:           "not found",
			oldPoolName:    "nonexistent",
			newPoolName:    "new-name",
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
				assert.Equal(t, "/v3/ips/pools/"+tt.oldPoolName, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.newPoolName, reqBody["name"])

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			client := NewSendGridClient("test-api-key", server.URL)

			reqBody := map[string]interface{}{
				"name": tt.newPoolName,
			}

			var result ipPoolAPIResponse
			err := client.Put(context.Background(), "/v3/ips/pools/"+tt.oldPoolName, reqBody, &result)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newPoolName, result.Name)
			}
		})
	}
}

func TestSendGridClient_DeleteIpPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		poolName       string
		responseStatus int
		expectError    bool
		expectNotFound bool
	}{
		{
			name:           "successful delete",
			poolName:       "marketing",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "not found (already deleted)",
			poolName:       "nonexistent",
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
				assert.Equal(t, "/v3/ips/pools/"+tt.poolName, r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.responseStatus)
			})

			client := NewSendGridClient("test-api-key", server.URL)

			err := client.Delete(context.Background(), "/v3/ips/pools/"+tt.poolName)

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

func TestIpPoolAPIResponse_ToState(t *testing.T) {
	t.Parallel()

	t.Run("with pool_name field", func(t *testing.T) {
		t.Parallel()

		resp := ipPoolAPIResponse{
			PoolName: "marketing",
			Ips:      []string{"167.89.21.3", "167.89.22.4"},
		}

		state := resp.toState()

		assert.Equal(t, "marketing", state.Name)
		assert.Equal(t, "marketing", state.PoolName)
		assert.Equal(t, []string{"167.89.21.3", "167.89.22.4"}, state.Ips)
	})

	t.Run("with name field (from update)", func(t *testing.T) {
		t.Parallel()

		resp := ipPoolAPIResponse{
			Name: "transactional",
			Ips:  nil,
		}

		state := resp.toState()

		assert.Equal(t, "transactional", state.Name)
		assert.Equal(t, "transactional", state.PoolName)
	})

	t.Run("with empty IPs", func(t *testing.T) {
		t.Parallel()

		resp := ipPoolAPIResponse{
			PoolName: "new-pool",
			Ips:      []string{},
		}

		state := resp.toState()

		assert.Equal(t, "new-pool", state.PoolName)
		assert.Empty(t, state.Ips)
	})
}

func TestIpPool_UrlEncoding(t *testing.T) {
	t.Parallel()

	// Test that pool names with special characters are properly handled
	t.Run("pool name with spaces", func(t *testing.T) {
		t.Parallel()

		server := mockSendGridServer(t, func(w http.ResponseWriter, r *http.Request) {
			// The path should contain URL-encoded space
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "/v3/ips/pools/")

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pool_name": "my pool", "ips": []}`))
		})

		client := NewSendGridClient("test-api-key", server.URL)

		var result ipPoolAPIResponse
		err := client.Get(context.Background(), "/v3/ips/pools/my%20pool", &result)

		require.NoError(t, err)
		assert.Equal(t, "my pool", result.PoolName)
	})
}
