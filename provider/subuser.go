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
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Subuser is the controller for the SendGrid Subuser resource.
//
// This resource manages SendGrid Subusers, which are separate accounts under
// a parent account that can be used to segment email sending and maintain
// separate sending reputations.
type Subuser struct{}

// SubuserArgs are the inputs to the Subuser resource.
type SubuserArgs struct {
	// Username is the username for the subuser (required)
	// This will be used as the resource ID
	Username string `pulumi:"username"`

	// Email is the email address of the subuser (required)
	Email string `pulumi:"email"`

	// Password is the password for the subuser to log into SendGrid (required)
	// This is only used during creation and cannot be retrieved
	Password string `pulumi:"password" provider:"secret"`

	// Ips is the list of IP addresses assigned to this subuser (optional)
	Ips []string `pulumi:"ips,optional"`

	// Region is the region this subuser should be assigned to (optional)
	// Valid values: "global" or "eu"
	// Requires SendGrid Pro plan or above
	Region *string `pulumi:"region,optional"`

	// Disabled indicates whether the subuser is disabled (optional)
	Disabled *bool `pulumi:"disabled,optional"`
}

// SubuserState is the state of the Subuser resource.
type SubuserState struct {
	// Username is the username of the subuser
	Username string `pulumi:"username"`

	// Email is the email address of the subuser
	Email string `pulumi:"email"`

	// UserID is the numeric ID assigned by SendGrid
	UserID int64 `pulumi:"userId"`

	// Ips is the list of IP addresses assigned to this subuser
	Ips []string `pulumi:"ips,optional"`

	// Region is the region this subuser is assigned to
	Region *string `pulumi:"region,optional"`

	// Disabled indicates whether the subuser is disabled
	Disabled bool `pulumi:"disabled"`
}

// Annotate provides descriptions for the Subuser resource.
func (s *Subuser) Annotate(annotator infer.Annotator) {
	annotator.Describe(&s, "Manages a SendGrid Subuser.\n\n"+
		"Subusers are separate accounts under a parent account that can be used to "+
		"segment email sending, maintain separate sending reputations, and organize "+
		"email workflows. Each subuser has their own credentials and can be assigned "+
		"specific IP addresses.\n\n"+
		"Note: The password is only used during creation and cannot be retrieved. "+
		"Regional subusers require a SendGrid Pro plan or above.")
}

// subuserCreateResponse represents the SendGrid API response for subuser creation
type subuserCreateResponse struct {
	UserID   int64    `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Ips      []string `json:"ips,omitempty"`
	Region   string   `json:"region,omitempty"`
}

// subuserGetResponse represents the SendGrid API response for getting a subuser
type subuserGetResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
}

// Create creates a new SendGrid Subuser.
func (s *Subuser) Create(ctx context.Context, req infer.CreateRequest[SubuserArgs]) (infer.CreateResponse[SubuserState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		disabled := false
		if input.Disabled != nil {
			disabled = *input.Disabled
		}
		state := SubuserState{
			Username: input.Username,
			Email:    input.Email,
			UserID:   0,
			Ips:      input.Ips,
			Region:   input.Region,
			Disabled: disabled,
		}
		return infer.CreateResponse[SubuserState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[SubuserState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"username": input.Username,
		"email":    input.Email,
		"password": input.Password,
	}
	if len(input.Ips) > 0 {
		reqBody["ips"] = input.Ips
	}
	if input.Region != nil {
		reqBody["region"] = *input.Region
		reqBody["include_region"] = true
	}

	// Make the API call to create subuser
	// POST /v3/subusers
	var result subuserCreateResponse
	if err := client.Post(ctx, "/v3/subusers", reqBody, &result); err != nil {
		return infer.CreateResponse[SubuserState]{}, fmt.Errorf("failed to create subuser: %w", err)
	}

	var region *string
	if result.Region != "" {
		region = &result.Region
	}

	state := SubuserState{
		Username: result.Username,
		Email:    result.Email,
		UserID:   result.UserID,
		Ips:      result.Ips,
		Region:   region,
		Disabled: false, // New subusers are enabled by default
	}

	// If disabled is requested, update the subuser to disable it
	if input.Disabled != nil && *input.Disabled {
		if err := s.setDisabled(ctx, client, input.Username, true); err != nil {
			return infer.CreateResponse[SubuserState]{}, fmt.Errorf("subuser created but failed to disable: %w", err)
		}
		state.Disabled = true
	}

	return infer.CreateResponse[SubuserState]{
		ID:     input.Username,
		Output: state,
	}, nil
}

// setDisabled enables or disables a subuser
func (s *Subuser) setDisabled(ctx context.Context, client *SendGridClient, username string, disabled bool) error {
	encodedUsername := url.PathEscape(username)
	reqBody := map[string]interface{}{
		"disabled": disabled,
	}
	return client.Patch(ctx, fmt.Sprintf("/v3/subusers/%s", encodedUsername), reqBody, nil)
}

// Read retrieves the current state of a SendGrid Subuser.
func (s *Subuser) Read(ctx context.Context, req infer.ReadRequest[SubuserArgs, SubuserState]) (infer.ReadResponse[SubuserArgs, SubuserState], error) {
	id := req.ID // id is the username
	oldState := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[SubuserArgs, SubuserState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL-encode the username
	encodedUsername := url.PathEscape(id)

	// Get subuser details
	// GET /v3/subusers/{subuser_name}
	var result subuserGetResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/subusers/%s", encodedUsername), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.ReadResponse[SubuserArgs, SubuserState]{}, nil
		}
		return infer.ReadResponse[SubuserArgs, SubuserState]{}, fmt.Errorf("failed to read subuser: %w", err)
	}

	state := SubuserState{
		Username: result.Username,
		Email:    result.Email,
		UserID:   result.ID,
		Disabled: result.Disabled,
		// Preserve IPs and Region from old state as they're not returned by GET
		Ips:    oldState.Ips,
		Region: oldState.Region,
	}

	inputs := SubuserArgs{
		Username: result.Username,
		Email:    result.Email,
		Ips:      oldState.Ips,
		Region:   oldState.Region,
		Disabled: &result.Disabled,
		// Password is not returned by the API; preserve the old input value to avoid perpetual diffs
		Password: req.Inputs.Password,
	}

	return infer.ReadResponse[SubuserArgs, SubuserState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Subuser.
func (s *Subuser) Update(ctx context.Context, req infer.UpdateRequest[SubuserArgs, SubuserState]) (infer.UpdateResponse[SubuserState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		disabled := oldState.Disabled
		if input.Disabled != nil {
			disabled = *input.Disabled
		}
		state := SubuserState{
			Username: oldState.Username,
			Email:    oldState.Email,
			UserID:   oldState.UserID,
			Ips:      input.Ips,
			Region:   input.Region,
			Disabled: disabled,
		}
		return infer.UpdateResponse[SubuserState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[SubuserState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL-encode the username
	encodedUsername := url.PathEscape(id)

	// Update disabled status if changed
	if input.Disabled != nil && *input.Disabled != oldState.Disabled {
		if err := s.setDisabled(ctx, client, id, *input.Disabled); err != nil {
			return infer.UpdateResponse[SubuserState]{}, fmt.Errorf("failed to update subuser disabled status: %w", err)
		}
	}

	// Update IPs if changed
	if len(input.Ips) > 0 && !stringSlicesEqual(input.Ips, oldState.Ips) {
		reqBody := map[string]interface{}{
			"ips": input.Ips,
		}
		if err := client.Put(ctx, fmt.Sprintf("/v3/subusers/%s/ips", encodedUsername), reqBody, nil); err != nil {
			return infer.UpdateResponse[SubuserState]{}, fmt.Errorf("failed to update subuser IPs: %w", err)
		}
	}

	disabled := oldState.Disabled
	if input.Disabled != nil {
		disabled = *input.Disabled
	}

	state := SubuserState{
		Username: oldState.Username,
		Email:    oldState.Email,
		UserID:   oldState.UserID,
		Ips:      input.Ips,
		Region:   input.Region,
		Disabled: disabled,
	}

	return infer.UpdateResponse[SubuserState]{Output: state}, nil
}

// Delete removes a SendGrid Subuser.
func (s *Subuser) Delete(ctx context.Context, req infer.DeleteRequest[SubuserState]) (infer.DeleteResponse, error) {
	id := req.ID // id is the username

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL-encode the username
	encodedUsername := url.PathEscape(id)

	// Delete the subuser
	// DELETE /v3/subusers/{subuser_name}
	if err := client.Delete(ctx, fmt.Sprintf("/v3/subusers/%s", encodedUsername)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete subuser: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
