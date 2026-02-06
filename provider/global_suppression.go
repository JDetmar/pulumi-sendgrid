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

// GlobalSuppression is the controller for the SendGrid Global Suppression resource.
//
// This resource manages email addresses in SendGrid's global suppression list.
// When an email address is globally suppressed, all emails to that address
// will be suppressed (not sent) across all unsubscribe groups.
type GlobalSuppression struct{}

// GlobalSuppressionArgs are the inputs to the GlobalSuppression resource.
type GlobalSuppressionArgs struct {
	// Email is the email address to add to the global suppression list (required)
	Email string `pulumi:"email"`
}

// GlobalSuppressionState is the state of the GlobalSuppression resource.
type GlobalSuppressionState struct {
	// Embed the input args in the output state
	GlobalSuppressionArgs

	// CreatedAt is the Unix timestamp when the suppression was created
	CreatedAt int64 `pulumi:"createdAt,optional"`
}

// Annotate provides descriptions for the GlobalSuppression resource.
func (g *GlobalSuppression) Annotate(annotator infer.Annotator) {
	annotator.Describe(&g, "Manages a SendGrid Global Suppression.\n\n"+
		"Global suppressions are email addresses that have been unsubscribed from all "+
		"types of emails. When an email address is globally suppressed, no emails will "+
		"be sent to that address regardless of the unsubscribe group.\n\n"+
		"This is useful for managing email addresses that have permanently opted out "+
		"of all communications, or for test addresses that should never receive emails.")
}

// Create adds an email address to the global suppression list.
func (g *GlobalSuppression) Create(ctx context.Context, req infer.CreateRequest[GlobalSuppressionArgs]) (infer.CreateResponse[GlobalSuppressionState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := GlobalSuppressionState{
			GlobalSuppressionArgs: input,
			CreatedAt:             0,
		}
		return infer.CreateResponse[GlobalSuppressionState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[GlobalSuppressionState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body - SendGrid expects an array of emails
	reqBody := map[string]interface{}{
		"recipient_emails": []string{input.Email},
	}

	// Make the API call to add to global suppressions
	// POST /v3/asm/suppressions/global
	var result struct {
		RecipientEmails []string `json:"recipient_emails"`
	}

	if err := client.Post(ctx, "/v3/asm/suppressions/global", reqBody, &result); err != nil {
		return infer.CreateResponse[GlobalSuppressionState]{}, fmt.Errorf("failed to add email to global suppression: %w", err)
	}

	// Verify the email was added
	found := false
	for _, email := range result.RecipientEmails {
		if email == input.Email {
			found = true
			break
		}
	}
	if !found {
		return infer.CreateResponse[GlobalSuppressionState]{}, fmt.Errorf("email was not added to global suppression list")
	}

	state := GlobalSuppressionState{
		GlobalSuppressionArgs: input,
		CreatedAt:             0, // SendGrid doesn't return created_at on creation
	}

	// Use the email as the resource ID
	return infer.CreateResponse[GlobalSuppressionState]{
		ID:     input.Email,
		Output: state,
	}, nil
}

// Read retrieves the current state of a global suppression.
func (g *GlobalSuppression) Read(ctx context.Context, req infer.ReadRequest[GlobalSuppressionArgs, GlobalSuppressionState]) (infer.ReadResponse[GlobalSuppressionArgs, GlobalSuppressionState], error) {
	id := req.ID // id is the email address

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[GlobalSuppressionArgs, GlobalSuppressionState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL-encode the email address for the path
	encodedEmail := url.PathEscape(id)

	// Check if the email is in the global suppression list
	// GET /v3/asm/suppressions/global/{email}
	// Returns {"recipient_email": "..."} (single object, not array)
	var result struct {
		RecipientEmail string `json:"recipient_email"`
	}

	if err := client.Get(ctx, fmt.Sprintf("/v3/asm/suppressions/global/%s", encodedEmail), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[GlobalSuppressionArgs, GlobalSuppressionState]{}, nil
		}
		return infer.ReadResponse[GlobalSuppressionArgs, GlobalSuppressionState]{}, fmt.Errorf("failed to read global suppression: %w", err)
	}

	// If the result is empty, the email is not suppressed
	if result.RecipientEmail == "" {
		return infer.ReadResponse[GlobalSuppressionArgs, GlobalSuppressionState]{}, nil
	}

	state := GlobalSuppressionState{
		GlobalSuppressionArgs: GlobalSuppressionArgs{
			Email: id,
		},
	}

	inputs := GlobalSuppressionArgs{
		Email: id,
	}

	return infer.ReadResponse[GlobalSuppressionArgs, GlobalSuppressionState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update is not supported for global suppressions since there's only one field (email).
// Changing the email would require deleting and recreating.
func (g *GlobalSuppression) Update(_ context.Context, _ infer.UpdateRequest[GlobalSuppressionArgs, GlobalSuppressionState]) (infer.UpdateResponse[GlobalSuppressionState], error) {
	// Global suppressions don't support updates - if the email changes, it's a replace
	// The Pulumi SDK handles this by checking for "replaceOnChanges" behavior
	return infer.UpdateResponse[GlobalSuppressionState]{}, fmt.Errorf("global suppressions cannot be updated - email changes require replacement")
}

// Delete removes an email address from the global suppression list.
func (g *GlobalSuppression) Delete(ctx context.Context, req infer.DeleteRequest[GlobalSuppressionState]) (infer.DeleteResponse, error) {
	id := req.ID // id is the email address

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL-encode the email address for the path
	encodedEmail := url.PathEscape(id)

	// Delete the global suppression
	// DELETE /v3/asm/suppressions/global/{email}
	if err := client.Delete(ctx, fmt.Sprintf("/v3/asm/suppressions/global/%s", encodedEmail)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete global suppression: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
