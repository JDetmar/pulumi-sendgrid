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
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ApiKey is the controller for the SendGrid API Key resource.
//
// This resource manages SendGrid API Keys, which are used to authenticate
// access to SendGrid services.
type ApiKey struct{} //nolint:revive // name matches Pulumi resource token

// ApiKeyArgs are the inputs to the ApiKey resource.
type ApiKeyArgs struct { //nolint:revive // name matches Pulumi resource token
	// Name is the name of the API key (required)
	Name string `pulumi:"name"`

	// Scopes is the list of permissions for this API key (optional).
	// If omitted, the key will have "Full Access" permissions by default.
	// See https://www.twilio.com/docs/sendgrid/api-reference/api-key-permissions/api-key-permissions
	// for available scopes.
	Scopes []string `pulumi:"scopes,optional"`
}

// ApiKeyState is the state of the ApiKey resource.
type ApiKeyState struct { //nolint:revive // name matches Pulumi resource token
	// Embed the input args in the output state
	ApiKeyArgs

	// APIKeyID is the unique identifier for this API key
	APIKeyID string `pulumi:"apiKeyId"`

	// APIKeyValue is the actual API key value. This is only returned on creation
	// and cannot be retrieved again, so it's marked as a secret and optional.
	// After creation, subsequent reads/updates won't have access to this value.
	APIKeyValue string `pulumi:"apiKeyValue,optional" provider:"secret"`
}

// Annotate provides descriptions and default values for the ApiKey resource.
func (a *ApiKey) Annotate(annotator infer.Annotator) {
	annotator.Describe(&a, "Manages a SendGrid API Key.\n\n"+
		"API keys are used to authenticate access to SendGrid services. "+
		"You can create keys with specific scopes to limit their permissions.\n\n"+
		"**Note:** The actual API key value is only returned on creation and cannot "+
		"be retrieved again. Make sure to store it securely.")
}

// Create creates a new SendGrid API Key.
func (a *ApiKey) Create(ctx context.Context, req infer.CreateRequest[ApiKeyArgs]) (infer.CreateResponse[ApiKeyState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := ApiKeyState{
			ApiKeyArgs:  input,
			APIKeyID:    "[computed]",
			APIKeyValue: "[computed]",
		}
		return infer.CreateResponse[ApiKeyState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[ApiKeyState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"name": input.Name,
	}
	if len(input.Scopes) > 0 {
		reqBody["scopes"] = input.Scopes
	}

	// Make the API call
	var result struct {
		APIKey   string   `json:"api_key"`
		APIKeyID string   `json:"api_key_id"`
		Name     string   `json:"name"`
		Scopes   []string `json:"scopes"`
	}

	if err := client.Post(ctx, "/v3/api_keys", reqBody, &result); err != nil {
		return infer.CreateResponse[ApiKeyState]{}, fmt.Errorf("failed to create API key: %w", err)
	}

	state := ApiKeyState{
		ApiKeyArgs: ApiKeyArgs{
			Name:   result.Name,
			Scopes: result.Scopes,
		},
		APIKeyID:    result.APIKeyID,
		APIKeyValue: result.APIKey,
	}

	return infer.CreateResponse[ApiKeyState]{
		ID:     result.APIKeyID,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid API Key.
func (a *ApiKey) Read(ctx context.Context, req infer.ReadRequest[ApiKeyArgs, ApiKeyState]) (infer.ReadResponse[ApiKeyArgs, ApiKeyState], error) {
	id := req.ID
	oldState := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the API key details
	var result struct {
		APIKeyID string   `json:"api_key_id"`
		Name     string   `json:"name"`
		Scopes   []string `json:"scopes"`
	}

	if err := client.Get(ctx, fmt.Sprintf("/v3/api_keys/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, nil
		}
		return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{}, fmt.Errorf("failed to read API key: %w", err)
	}

	// Update state with values from API
	state := ApiKeyState{
		ApiKeyArgs: ApiKeyArgs{
			Name:   result.Name,
			Scopes: result.Scopes,
		},
		APIKeyID: result.APIKeyID,
		// Preserve the API key from old state since it can't be retrieved
		APIKeyValue: oldState.APIKeyValue,
	}

	inputs := ApiKeyArgs{
		Name:   result.Name,
		Scopes: result.Scopes,
	}

	return infer.ReadResponse[ApiKeyArgs, ApiKeyState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid API Key.
func (a *ApiKey) Update(ctx context.Context, req infer.UpdateRequest[ApiKeyArgs, ApiKeyState]) (infer.UpdateResponse[ApiKeyState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := ApiKeyState{
			ApiKeyArgs:  input,
			APIKeyID:    oldState.APIKeyID,
			APIKeyValue: oldState.APIKeyValue,
		}
		return infer.UpdateResponse[ApiKeyState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[ApiKeyState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Use PUT to update both name and scopes
	reqBody := map[string]interface{}{
		"name": input.Name,
	}
	if len(input.Scopes) > 0 {
		reqBody["scopes"] = input.Scopes
	} else {
		// If no scopes provided, we need to fetch current scopes or set empty
		reqBody["scopes"] = []string{}
	}

	var result struct {
		APIKeyID string   `json:"api_key_id"`
		Name     string   `json:"name"`
		Scopes   []string `json:"scopes"`
	}

	if err := client.Put(ctx, fmt.Sprintf("/v3/api_keys/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[ApiKeyState]{}, fmt.Errorf("failed to update API key: %w", err)
	}

	state := ApiKeyState{
		ApiKeyArgs: ApiKeyArgs{
			Name:   result.Name,
			Scopes: result.Scopes,
		},
		APIKeyID: result.APIKeyID,
		// Preserve the API key from old state since it can't be retrieved
		APIKeyValue: oldState.APIKeyValue,
	}

	return infer.UpdateResponse[ApiKeyState]{Output: state}, nil
}

// Delete removes a SendGrid API Key.
func (a *ApiKey) Delete(ctx context.Context, req infer.DeleteRequest[ApiKeyState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/api_keys/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete API key: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
