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

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"

	sendgrid "github.com/JDetmar/pulumi-sendgrid/provider"
)

// skipIfNoApiKey skips the test if SENDGRID_API_KEY is not set
func skipIfNoApiKey(t *testing.T) string {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		t.Skip("SENDGRID_API_KEY environment variable not set, skipping integration test")
	}
	return apiKey
}

// apiKeyUrn returns a URN for an ApiKey resource
func apiKeyUrn(name string) resource.URN {
	return resource.NewURN("integration", "sendgrid-test", "",
		tokens.Type("sendgrid:index:ApiKey"), name)
}

// configuredProvider creates a provider with SendGrid API key configured
func configuredProvider(t *testing.T, apiKey string) integration.Server {
	s, err := integration.NewServer(
		context.Background(),
		sendgrid.Name,
		semver.MustParse("1.0.0"),
		integration.WithProvider(sendgrid.Provider()),
	)
	require.NoError(t, err)

	// Configure the provider with the API key
	err = s.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			"apiKey": property.New(apiKey),
		}),
	})
	require.NoError(t, err)

	return s
}

// sendGridAPIGet performs a GET request to the SendGrid API
func sendGridAPIGet(apiKey, path string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.sendgrid.com"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// newArrayValue creates a property.Value containing an array
func newArrayValue(values ...string) property.Value {
	arr := make([]property.Value, len(values))
	for i, v := range values {
		arr[i] = property.New(v)
	}
	return property.New(property.NewArray(arr))
}

// TestApiKeyIntegration_FullLifecycle tests the full CRUD lifecycle of an API key
func TestApiKeyIntegration_FullLifecycle(t *testing.T) {
	apiKey := skipIfNoApiKey(t)
	prov := configuredProvider(t, apiKey)

	testKeyName := fmt.Sprintf("pulumi-integration-test-%d", time.Now().Unix())
	testKeyNameUpdated := testKeyName + "-updated"

	var createdKeyID string

	// Cleanup function to ensure the key is deleted even if test fails
	t.Cleanup(func() {
		if createdKeyID != "" {
			// Try to delete the key if it still exists
			_ = prov.Delete(p.DeleteRequest{
				Urn: apiKeyUrn("test-key"),
				ID:  createdKeyID,
				Properties: property.NewMap(map[string]property.Value{
					"name":     property.New(testKeyNameUpdated),
					"scopes":   newArrayValue("mail.send"),
					"apiKeyId": property.New(createdKeyID),
				}),
			})
		}
	})

	// Step 1: Create API Key
	t.Run("Create", func(t *testing.T) {
		createResp, err := prov.Create(p.CreateRequest{
			Urn: apiKeyUrn("test-key"),
			Properties: property.NewMap(map[string]property.Value{
				"name":   property.New(testKeyName),
				"scopes": newArrayValue("mail.send"),
			}),
			DryRun: false,
		})
		require.NoError(t, err)
		require.NotEmpty(t, createResp.ID)

		createdKeyID = createResp.ID

		// Verify the returned properties
		props := createResp.Properties
		assert.Equal(t, testKeyName, props.Get("name").AsString())
		assert.NotEmpty(t, props.Get("apiKey").AsString(), "API key value should be returned on creation")
		assert.Equal(t, createdKeyID, props.Get("apiKeyId").AsString())

		t.Logf("Created API key with ID: %s", createdKeyID)
	})

	// Step 2: Verify via direct API call
	t.Run("VerifyCreation", func(t *testing.T) {
		if createdKeyID == "" {
			t.Skip("No key was created")
		}

		result, err := sendGridAPIGet(apiKey, "/v3/api_keys/"+createdKeyID)
		require.NoError(t, err)

		assert.Equal(t, createdKeyID, result["api_key_id"])
		assert.Equal(t, testKeyName, result["name"])
		t.Logf("Verified API key exists in SendGrid: %v", result)
	})

	// Step 3: Read API Key
	t.Run("Read", func(t *testing.T) {
		if createdKeyID == "" {
			t.Skip("No key was created")
		}

		readResp, err := prov.Read(p.ReadRequest{
			Urn: apiKeyUrn("test-key"),
			ID:  createdKeyID,
			Properties: property.NewMap(map[string]property.Value{
				"name":     property.New(testKeyName),
				"scopes":   newArrayValue("mail.send"),
				"apiKeyId": property.New(createdKeyID),
			}),
		})
		require.NoError(t, err)
		assert.Equal(t, createdKeyID, readResp.ID)
		assert.Equal(t, testKeyName, readResp.Properties.Get("name").AsString())
		t.Logf("Read API key state: name=%s", readResp.Properties.Get("name").AsString())
	})

	// Step 4: Update API Key
	t.Run("Update", func(t *testing.T) {
		if createdKeyID == "" {
			t.Skip("No key was created")
		}

		updateResp, err := prov.Update(p.UpdateRequest{
			Urn: apiKeyUrn("test-key"),
			ID:  createdKeyID,
			State: property.NewMap(map[string]property.Value{
				"name":     property.New(testKeyName),
				"scopes":   newArrayValue("mail.send"),
				"apiKeyId": property.New(createdKeyID),
			}),
			Inputs: property.NewMap(map[string]property.Value{
				"name":   property.New(testKeyNameUpdated),
				"scopes": newArrayValue("mail.send", "alerts.read"),
			}),
			DryRun: false,
		})
		require.NoError(t, err)

		assert.Equal(t, testKeyNameUpdated, updateResp.Properties.Get("name").AsString())
		t.Logf("Updated API key name to: %s", testKeyNameUpdated)
	})

	// Step 5: Verify update via direct API call
	t.Run("VerifyUpdate", func(t *testing.T) {
		if createdKeyID == "" {
			t.Skip("No key was created")
		}

		result, err := sendGridAPIGet(apiKey, "/v3/api_keys/"+createdKeyID)
		require.NoError(t, err)

		assert.Equal(t, testKeyNameUpdated, result["name"])
		t.Logf("Verified API key update in SendGrid: name=%s", result["name"])
	})

	// Step 6: Delete API Key
	t.Run("Delete", func(t *testing.T) {
		if createdKeyID == "" {
			t.Skip("No key was created")
		}

		keyIDToDelete := createdKeyID

		err := prov.Delete(p.DeleteRequest{
			Urn: apiKeyUrn("test-key"),
			ID:  keyIDToDelete,
			Properties: property.NewMap(map[string]property.Value{
				"name":     property.New(testKeyNameUpdated),
				"scopes":   newArrayValue("mail.send", "alerts.read"),
				"apiKeyId": property.New(keyIDToDelete),
			}),
		})
		require.NoError(t, err)
		t.Logf("Deleted API key: %s", keyIDToDelete)

		// Mark as deleted so cleanup doesn't try again
		createdKeyID = ""
	})

	// Step 7: Verify deletion via direct API call
	t.Run("VerifyDeletion", func(t *testing.T) {
		// Give SendGrid a moment to process the deletion
		time.Sleep(1 * time.Second)

		result, err := sendGridAPIGet(apiKey, "/v3/api_keys")
		require.NoError(t, err)

		// Check that our key is no longer in the list
		keys, ok := result["result"].([]interface{})
		require.True(t, ok)

		for _, k := range keys {
			key := k.(map[string]interface{})
			assert.NotEqual(t, testKeyName, key["name"], "Deleted key should not exist")
			assert.NotEqual(t, testKeyNameUpdated, key["name"], "Deleted key should not exist")
		}
		t.Logf("Verified API key no longer exists in SendGrid")
	})
}

// TestApiKeyIntegration_Preview tests that preview (dry run) doesn't create resources
func TestApiKeyIntegration_Preview(t *testing.T) {
	apiKey := skipIfNoApiKey(t)
	prov := configuredProvider(t, apiKey)

	testKeyName := fmt.Sprintf("pulumi-preview-test-%d", time.Now().Unix())

	// Preview create (dry run)
	createResp, err := prov.Create(p.CreateRequest{
		Urn: apiKeyUrn("preview-key"),
		Properties: property.NewMap(map[string]property.Value{
			"name":   property.New(testKeyName),
			"scopes": newArrayValue("mail.send"),
		}),
		DryRun: true,
	})
	require.NoError(t, err)

	// Preview should return placeholder values
	assert.Equal(t, "[preview]", createResp.ID)
	// The apiKeyId might be returned as a computed value or a placeholder string
	apiKeyIdVal := createResp.Properties.Get("apiKeyId")
	assert.True(t, apiKeyIdVal.IsComputed() || apiKeyIdVal.AsString() == "[computed]",
		"Preview should return computed or placeholder apiKeyId")
	t.Logf("Preview returned placeholder ID as expected")

	// Verify no key was actually created
	result, err := sendGridAPIGet(apiKey, "/v3/api_keys")
	require.NoError(t, err)

	keys, ok := result["result"].([]interface{})
	require.True(t, ok)

	for _, k := range keys {
		key := k.(map[string]interface{})
		assert.NotEqual(t, testKeyName, key["name"], "Preview should not create actual resources")
	}
	t.Logf("Verified no resource was created during preview")
}

// TestApiKeyIntegration_ReadNotFound tests that reading a non-existent key returns empty
func TestApiKeyIntegration_ReadNotFound(t *testing.T) {
	apiKey := skipIfNoApiKey(t)
	prov := configuredProvider(t, apiKey)

	readResp, err := prov.Read(p.ReadRequest{
		Urn: apiKeyUrn("nonexistent-key"),
		ID:  "nonexistent-id-12345",
		Properties: property.NewMap(map[string]property.Value{
			"name":     property.New("nonexistent"),
			"apiKeyId": property.New("nonexistent-id-12345"),
		}),
	})

	require.NoError(t, err)
	// When a resource doesn't exist, Read should return empty ID to signal deletion
	assert.Empty(t, readResp.ID, "Reading non-existent resource should return empty ID")
	t.Logf("Read of non-existent key correctly returned empty response")
}
