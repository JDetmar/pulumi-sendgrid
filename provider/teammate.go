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

// Teammate is the controller for the SendGrid Teammate resource.
//
// This resource manages SendGrid Teammates, which are users that have access
// to your SendGrid account with configurable permissions.
type Teammate struct{}

// TeammateArgs are the inputs to the Teammate resource.
type TeammateArgs struct {
	// Email is the email address of the teammate to invite (required)
	Email string `pulumi:"email"`

	// Scopes is the list of permissions for this teammate (optional)
	// See https://docs.sendgrid.com/api-reference/how-to-use-the-sendgrid-v3-api/authorization
	// for available scopes.
	Scopes []string `pulumi:"scopes,optional"`

	// IsAdmin indicates whether the teammate should have full admin access (optional)
	// When true, the teammate has all permissions
	IsAdmin *bool `pulumi:"isAdmin,optional"`
}

// TeammateState is the state of the Teammate resource.
type TeammateState struct {
	// Email is the email address of the teammate
	Email string `pulumi:"email"`

	// Scopes is the list of permissions for this teammate
	Scopes []string `pulumi:"scopes,optional"`

	// IsAdmin indicates whether the teammate has admin access
	IsAdmin bool `pulumi:"isAdmin"`

	// Username is the username assigned after the teammate accepts the invitation
	// This will be empty until the invitation is accepted
	Username string `pulumi:"username,optional"`

	// FirstName is the teammate's first name (populated after accepting invite)
	FirstName string `pulumi:"firstName,optional"`

	// LastName is the teammate's last name (populated after accepting invite)
	LastName string `pulumi:"lastName,optional"`

	// UserType indicates the type of user (teammate, admin, etc.)
	UserType string `pulumi:"userType,optional"`

	// Token is the invitation token (available for pending invitations)
	Token string `pulumi:"token,optional"`
}

// Annotate provides descriptions for the Teammate resource.
func (t *Teammate) Annotate(annotator infer.Annotator) {
	annotator.Describe(&t, "Manages a SendGrid Teammate.\n\n"+
		"Teammates are users who have access to your SendGrid account with configurable "+
		"permissions. You can invite teammates via email and set their initial permissions "+
		"using scopes.\n\n"+
		"Note: Teammate invitations expire after 7 days. The invitation can be resent "+
		"to reset the expiration. Free and Essentials plans allow only one teammate per account.")
}

// teammateInviteResponse represents the SendGrid API response for teammate invitation
type teammateInviteResponse struct {
	Email   string   `json:"email"`
	Scopes  []string `json:"scopes,omitempty"`
	IsAdmin bool     `json:"is_admin"`
	Token   string   `json:"token"`
}

// teammateGetResponse represents the SendGrid API response for getting a teammate
type teammateGetResponse struct {
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	FirstName string   `json:"first_name,omitempty"`
	LastName  string   `json:"last_name,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	UserType  string   `json:"user_type"`
	IsAdmin   bool     `json:"is_admin"`
}

// Create creates a new SendGrid Teammate (sends invitation).
func (t *Teammate) Create(ctx context.Context, req infer.CreateRequest[TeammateArgs]) (infer.CreateResponse[TeammateState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		isAdmin := false
		if input.IsAdmin != nil {
			isAdmin = *input.IsAdmin
		}
		state := TeammateState{
			Email:   input.Email,
			Scopes:  input.Scopes,
			IsAdmin: isAdmin,
			Token:   "[computed]",
		}
		return infer.CreateResponse[TeammateState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[TeammateState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"email": input.Email,
	}
	if len(input.Scopes) > 0 {
		reqBody["scopes"] = input.Scopes
	}
	if input.IsAdmin != nil {
		reqBody["is_admin"] = *input.IsAdmin
	}

	// Make the API call to invite teammate
	// POST /v3/teammates
	var result teammateInviteResponse
	if err := client.Post(ctx, "/v3/teammates", reqBody, &result); err != nil {
		return infer.CreateResponse[TeammateState]{}, fmt.Errorf("failed to invite teammate: %w", err)
	}

	state := TeammateState{
		Email:   result.Email,
		Scopes:  result.Scopes,
		IsAdmin: result.IsAdmin,
		Token:   result.Token,
	}

	// Use email as the resource ID since username isn't assigned until invite is accepted
	return infer.CreateResponse[TeammateState]{
		ID:     input.Email,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Teammate.
func (t *Teammate) Read(ctx context.Context, req infer.ReadRequest[TeammateArgs, TeammateState]) (infer.ReadResponse[TeammateArgs, TeammateState], error) {
	id := req.ID // id is the email
	oldState := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[TeammateArgs, TeammateState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// First try to find the teammate by username if we have one
	if oldState.Username != "" {
		encodedUsername := url.PathEscape(oldState.Username)
		var result teammateGetResponse
		if err := client.Get(ctx, fmt.Sprintf("/v3/teammates/%s", encodedUsername), &result); err != nil {
			if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
				// Teammate was deleted
				return infer.ReadResponse[TeammateArgs, TeammateState]{}, nil
			}
			return infer.ReadResponse[TeammateArgs, TeammateState]{}, fmt.Errorf("failed to read teammate: %w", err)
		}

		state := TeammateState{
			Email:     result.Email,
			Scopes:    result.Scopes,
			IsAdmin:   result.IsAdmin,
			Username:  result.Username,
			FirstName: result.FirstName,
			LastName:  result.LastName,
			UserType:  result.UserType,
		}

		inputs := TeammateArgs{
			Email:   result.Email,
			Scopes:  result.Scopes,
			IsAdmin: &result.IsAdmin,
		}

		return infer.ReadResponse[TeammateArgs, TeammateState]{
			ID:     id,
			Inputs: inputs,
			State:  state,
		}, nil
	}

	// If no username, check pending invitations
	// The API returns {"result": [...]} not a bare array
	var pendingWrapper struct {
		Result []struct {
			Email   string   `json:"email"`
			Scopes  []string `json:"scopes,omitempty"`
			IsAdmin bool     `json:"is_admin"`
			Token   string   `json:"token"`
		} `json:"result"`
	}

	if err := client.Get(ctx, "/v3/teammates/pending", &pendingWrapper); err != nil {
		return infer.ReadResponse[TeammateArgs, TeammateState]{}, fmt.Errorf("failed to read pending teammates: %w", err)
	}
	pendingList := pendingWrapper.Result

	// Look for the pending invitation by email
	for _, pending := range pendingList {
		if pending.Email == id {
			state := TeammateState{
				Email:   pending.Email,
				Scopes:  pending.Scopes,
				IsAdmin: pending.IsAdmin,
				Token:   pending.Token,
			}

			inputs := TeammateArgs{
				Email:   pending.Email,
				Scopes:  pending.Scopes,
				IsAdmin: &pending.IsAdmin,
			}

			return infer.ReadResponse[TeammateArgs, TeammateState]{
				ID:     id,
				Inputs: inputs,
				State:  state,
			}, nil
		}
	}

	// Also check active teammates by listing all
	// The API returns {"result": [...]} not a bare array
	var teammatesWrapper struct {
		Result []teammateGetResponse `json:"result"`
	}
	if err := client.Get(ctx, "/v3/teammates", &teammatesWrapper); err != nil {
		return infer.ReadResponse[TeammateArgs, TeammateState]{}, fmt.Errorf("failed to list teammates: %w", err)
	}
	teammatesList := teammatesWrapper.Result

	for _, teammate := range teammatesList {
		if teammate.Email == id {
			state := TeammateState{
				Email:     teammate.Email,
				Scopes:    teammate.Scopes,
				IsAdmin:   teammate.IsAdmin,
				Username:  teammate.Username,
				FirstName: teammate.FirstName,
				LastName:  teammate.LastName,
				UserType:  teammate.UserType,
			}

			inputs := TeammateArgs{
				Email:   teammate.Email,
				Scopes:  teammate.Scopes,
				IsAdmin: &teammate.IsAdmin,
			}

			return infer.ReadResponse[TeammateArgs, TeammateState]{
				ID:     id,
				Inputs: inputs,
				State:  state,
			}, nil
		}
	}

	// Not found in pending or active
	return infer.ReadResponse[TeammateArgs, TeammateState]{}, nil
}

// Update updates an existing SendGrid Teammate.
func (t *Teammate) Update(ctx context.Context, req infer.UpdateRequest[TeammateArgs, TeammateState]) (infer.UpdateResponse[TeammateState], error) {
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		isAdmin := oldState.IsAdmin
		if input.IsAdmin != nil {
			isAdmin = *input.IsAdmin
		}
		state := TeammateState{
			Email:     oldState.Email,
			Scopes:    input.Scopes,
			IsAdmin:   isAdmin,
			Username:  oldState.Username,
			FirstName: oldState.FirstName,
			LastName:  oldState.LastName,
			UserType:  oldState.UserType,
			Token:     oldState.Token,
		}
		return infer.UpdateResponse[TeammateState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[TeammateState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Can only update scopes if the teammate has accepted the invitation
	if oldState.Username == "" {
		// For pending invitations, we can't update - return current state
		return infer.UpdateResponse[TeammateState]{Output: oldState}, nil
	}

	// Update teammate scopes
	// PATCH /v3/teammates/{username}
	encodedUsername := url.PathEscape(oldState.Username)
	reqBody := map[string]interface{}{}
	if len(input.Scopes) > 0 {
		reqBody["scopes"] = input.Scopes
	}
	if input.IsAdmin != nil {
		reqBody["is_admin"] = *input.IsAdmin
	}

	var result teammateGetResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/teammates/%s", encodedUsername), reqBody, &result); err != nil {
		return infer.UpdateResponse[TeammateState]{}, fmt.Errorf("failed to update teammate: %w", err)
	}

	state := TeammateState{
		Email:     result.Email,
		Scopes:    result.Scopes,
		IsAdmin:   result.IsAdmin,
		Username:  result.Username,
		FirstName: result.FirstName,
		LastName:  result.LastName,
		UserType:  result.UserType,
	}

	return infer.UpdateResponse[TeammateState]{Output: state}, nil
}

// Delete removes a SendGrid Teammate.
func (t *Teammate) Delete(ctx context.Context, req infer.DeleteRequest[TeammateState]) (infer.DeleteResponse, error) {
	state := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// If teammate has accepted invitation, delete by username
	if state.Username != "" {
		encodedUsername := url.PathEscape(state.Username)
		if err := client.Delete(ctx, fmt.Sprintf("/v3/teammates/%s", encodedUsername)); err != nil {
			if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
				return infer.DeleteResponse{}, nil
			}
			return infer.DeleteResponse{}, fmt.Errorf("failed to delete teammate: %w", err)
		}
		return infer.DeleteResponse{}, nil
	}

	// If pending invitation, delete by token
	if state.Token != "" {
		if err := client.Delete(ctx, fmt.Sprintf("/v3/teammates/pending/%s", state.Token)); err != nil {
			if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
				return infer.DeleteResponse{}, nil
			}
			return infer.DeleteResponse{}, fmt.Errorf("failed to delete pending teammate invitation: %w", err)
		}
		return infer.DeleteResponse{}, nil
	}

	// Neither username nor token - nothing to delete
	return infer.DeleteResponse{}, nil
}
