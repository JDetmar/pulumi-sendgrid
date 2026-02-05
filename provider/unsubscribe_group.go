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
	"strconv"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// UnsubscribeGroup is the controller for the SendGrid Unsubscribe Group resource.
//
// This resource manages SendGrid Unsubscribe Groups (ASM - Advanced Suppression Management),
// which allow recipients to opt out of specific types of emails while still receiving others.
type UnsubscribeGroup struct{}

// UnsubscribeGroupArgs are the inputs to the UnsubscribeGroup resource.
type UnsubscribeGroupArgs struct {
	// Name is the name of the unsubscribe group (required, max 30 chars)
	Name string `pulumi:"name"`

	// Description is a description of the unsubscribe group (optional, max 100 chars)
	Description *string `pulumi:"description,optional"`

	// IsDefault indicates whether this is the default unsubscribe group (optional)
	// When true, this group is used when no other group is specified
	IsDefault *bool `pulumi:"isDefault,optional"`
}

// UnsubscribeGroupState is the state of the UnsubscribeGroup resource.
type UnsubscribeGroupState struct {
	// Embed the input args in the output state
	UnsubscribeGroupArgs

	// GroupId is the ID assigned by SendGrid (returned from API)
	GroupId int `pulumi:"groupId"`

	// Unsubscribes is the count of emails that have been unsubscribed from this group
	Unsubscribes int `pulumi:"unsubscribes"`
}

// Annotate provides descriptions for the UnsubscribeGroup resource.
func (g *UnsubscribeGroup) Annotate(annotator infer.Annotator) {
	annotator.Describe(&g, "Manages a SendGrid Unsubscribe Group (Advanced Suppression Management).\n\n"+
		"Unsubscribe groups allow recipients to opt out of specific types of emails while "+
		"still receiving others. For example, you might have separate groups for marketing, "+
		"newsletters, and product updates, allowing users to choose which types of emails "+
		"they want to receive.\n\n"+
		"When a recipient unsubscribes from a group, they will no longer receive emails "+
		"that are associated with that group.")
}

// unsubscribeGroupAPIResponse represents the SendGrid API response structure for unsubscribe groups
type unsubscribeGroupAPIResponse struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsDefault    bool   `json:"is_default"`
	Unsubscribes int    `json:"unsubscribes"`
}

// toState converts an API response to UnsubscribeGroupState
func (r *unsubscribeGroupAPIResponse) toState() UnsubscribeGroupState {
	var description *string
	if r.Description != "" {
		description = &r.Description
	}

	isDefault := r.IsDefault

	return UnsubscribeGroupState{
		UnsubscribeGroupArgs: UnsubscribeGroupArgs{
			Name:        r.Name,
			Description: description,
			IsDefault:   &isDefault,
		},
		GroupId:      r.ID,
		Unsubscribes: r.Unsubscribes,
	}
}

// Create creates a new SendGrid Unsubscribe Group.
func (g *UnsubscribeGroup) Create(ctx context.Context, req infer.CreateRequest[UnsubscribeGroupArgs]) (infer.CreateResponse[UnsubscribeGroupState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		isDefault := false
		if input.IsDefault != nil {
			isDefault = *input.IsDefault
		}
		state := UnsubscribeGroupState{
			UnsubscribeGroupArgs: input,
			GroupId:              0,
			Unsubscribes:         0,
		}
		if state.IsDefault == nil {
			state.IsDefault = &isDefault
		}
		return infer.CreateResponse[UnsubscribeGroupState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[UnsubscribeGroupState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"name": input.Name,
	}
	if input.Description != nil {
		reqBody["description"] = *input.Description
	}
	if input.IsDefault != nil {
		reqBody["is_default"] = *input.IsDefault
	}

	// Make the API call
	var result unsubscribeGroupAPIResponse
	if err := client.Post(ctx, "/v3/asm/groups", reqBody, &result); err != nil {
		return infer.CreateResponse[UnsubscribeGroupState]{}, fmt.Errorf("failed to create unsubscribe group: %w", err)
	}

	state := result.toState()

	// Use the group ID as the Pulumi resource ID
	return infer.CreateResponse[UnsubscribeGroupState]{
		ID:     strconv.Itoa(result.ID),
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Unsubscribe Group.
func (g *UnsubscribeGroup) Read(ctx context.Context, req infer.ReadRequest[UnsubscribeGroupArgs, UnsubscribeGroupState]) (infer.ReadResponse[UnsubscribeGroupArgs, UnsubscribeGroupState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[UnsubscribeGroupArgs, UnsubscribeGroupState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the unsubscribe group details
	var result unsubscribeGroupAPIResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/asm/groups/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[UnsubscribeGroupArgs, UnsubscribeGroupState]{}, nil
		}
		return infer.ReadResponse[UnsubscribeGroupArgs, UnsubscribeGroupState]{}, fmt.Errorf("failed to read unsubscribe group: %w", err)
	}

	state := result.toState()
	inputs := state.UnsubscribeGroupArgs

	return infer.ReadResponse[UnsubscribeGroupArgs, UnsubscribeGroupState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Unsubscribe Group.
func (g *UnsubscribeGroup) Update(ctx context.Context, req infer.UpdateRequest[UnsubscribeGroupArgs, UnsubscribeGroupState]) (infer.UpdateResponse[UnsubscribeGroupState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := UnsubscribeGroupState{
			UnsubscribeGroupArgs: input,
			GroupId:              oldState.GroupId,
			Unsubscribes:         oldState.Unsubscribes,
		}
		return infer.UpdateResponse[UnsubscribeGroupState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[UnsubscribeGroupState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body - PATCH requires name and optionally description
	reqBody := map[string]interface{}{
		"name": input.Name,
	}
	if input.Description != nil {
		reqBody["description"] = *input.Description
	}
	// Note: is_default can only be changed via a separate API call or during creation
	// The PATCH endpoint does not support changing is_default

	// Make the API call (PATCH to update the group)
	var result unsubscribeGroupAPIResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/asm/groups/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[UnsubscribeGroupState]{}, fmt.Errorf("failed to update unsubscribe group: %w", err)
	}

	state := result.toState()

	// Preserve the IsDefault value from input if the API doesn't return it in PATCH response
	if input.IsDefault != nil {
		state.IsDefault = input.IsDefault
	}

	return infer.UpdateResponse[UnsubscribeGroupState]{Output: state}, nil
}

// Delete removes a SendGrid Unsubscribe Group.
func (g *UnsubscribeGroup) Delete(ctx context.Context, req infer.DeleteRequest[UnsubscribeGroupState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/asm/groups/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete unsubscribe group: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
