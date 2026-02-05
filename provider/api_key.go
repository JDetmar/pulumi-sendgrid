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

// APIKey is the controller for the SendGrid API Key resource.
//
// This resource manages SendGrid API Keys, which are used to authenticate
// access to SendGrid services.
type APIKey struct{}

// APIKeyArgs are the inputs to the APIKey resource.
type APIKeyArgs struct {
	// Name is the name of the API key (required)
	Name string `pulumi:"name"`

	// Scopes is the list of permissions for this API key (optional).
	// If omitted, the key will have "Full Access" permissions by default.
	// See https://docs.sendgrid.com/api-reference/how-to-use-the-sendgrid-v3-api/authorization
	// for available scopes.
	Scopes []string `pulumi:"scopes,optional"`
}

// APIKeyState is the state of the APIKey resource.
type APIKeyState struct {
	// Embed the input args in the output state
	APIKeyArgs

	// APIKeyID is the unique identifier for this API key
	APIKeyID string `pulumi:"apiKeyId"`

	// APIKey is the actual API key value. This is only returned on creation
	// and cannot be retrieved again, so it's marked as a secret and optional.
	// After creation, subsequent reads/updates won't have access to this value.
	APIKey string `pulumi:"apiKey,optional" provider:"secret"`
}

// Annotate provides descriptions and default values for the APIKey resource.
func (a *APIKey) Annotate(annotator infer.Annotator) {
	annotator.Describe(&a, "Manages a SendGrid API Key.\n\n"+
		"API keys are used to authenticate access to SendGrid services. "+
		"You can create keys with specific scopes to limit their permissions.\n\n"+
		"**Note:** The actual API key value is only returned on creation and cannot "+
		"be retrieved again. Make sure to store it securely.")
}

// Create creates a new SendGrid API Key.
func (a *APIKey) Create(ctx context.Context, req infer.CreateRequest[APIKeyArgs]) (infer.CreateResponse[APIKeyState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := APIKeyState{
			APIKeyArgs: input,
			APIKeyID:   "[computed]",
			APIKey:     "[computed]",
		}
		return infer.CreateResponse[APIKeyState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[APIKeyState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
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
		return infer.CreateResponse[APIKeyState]{}, fmt.Errorf("failed to create API key: %w", err)
	}

	state := APIKeyState{
		APIKeyArgs: APIKeyArgs{
			Name:   result.Name,
			Scopes: result.Scopes,
		},
		APIKeyID: result.APIKeyID,
		APIKey:   result.APIKey,
	}

	return infer.CreateResponse[APIKeyState]{
		ID:     result.APIKeyID,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid API Key.
func (a *APIKey) Read(ctx context.Context, req infer.ReadRequest[APIKeyArgs, APIKeyState]) (infer.ReadResponse[APIKeyArgs, APIKeyState], error) {
	id := req.ID
	oldState := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[APIKeyArgs, APIKeyState]{}, fmt.Errorf("SendGrid client not configured")
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
			return infer.ReadResponse[APIKeyArgs, APIKeyState]{}, nil
		}
		return infer.ReadResponse[APIKeyArgs, APIKeyState]{}, fmt.Errorf("failed to read API key: %w", err)
	}

	// Update state with values from API
	state := APIKeyState{
		APIKeyArgs: APIKeyArgs{
			Name:   result.Name,
			Scopes: result.Scopes,
		},
		APIKeyID: result.APIKeyID,
		// Preserve the API key from old state since it can't be retrieved
		APIKey: oldState.APIKey,
	}

	inputs := APIKeyArgs{
		Name:   result.Name,
		Scopes: result.Scopes,
	}

	return infer.ReadResponse[APIKeyArgs, APIKeyState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid API Key.
func (a *APIKey) Update(ctx context.Context, req infer.UpdateRequest[APIKeyArgs, APIKeyState]) (infer.UpdateResponse[APIKeyState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := APIKeyState{
			APIKeyArgs: input,
			APIKeyID:   oldState.APIKeyID,
			APIKey:     oldState.APIKey,
		}
		return infer.UpdateResponse[APIKeyState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[APIKeyState]{}, fmt.Errorf("SendGrid client not configured")
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
		return infer.UpdateResponse[APIKeyState]{}, fmt.Errorf("failed to update API key: %w", err)
	}

	state := APIKeyState{
		APIKeyArgs: APIKeyArgs{
			Name:   result.Name,
			Scopes: result.Scopes,
		},
		APIKeyID: result.APIKeyID,
		// Preserve the API key from old state since it can't be retrieved
		APIKey: oldState.APIKey,
	}

	return infer.UpdateResponse[APIKeyState]{Output: state}, nil
}

// Delete removes a SendGrid API Key.
func (a *APIKey) Delete(ctx context.Context, req infer.DeleteRequest[APIKeyState]) (infer.DeleteResponse, error) {
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
