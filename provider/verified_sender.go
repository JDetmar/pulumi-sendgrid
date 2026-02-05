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

// VerifiedSender is the controller for the SendGrid Verified Sender resource.
//
// This resource manages SendGrid Verified Senders, which are used to verify
// sender identities for email delivery.
type VerifiedSender struct{}

// VerifiedSenderArgs are the inputs to the VerifiedSender resource.
type VerifiedSenderArgs struct {
	// Nickname is a label for the sender identity (required)
	Nickname string `pulumi:"nickname"`

	// FromEmail is the email address to send from (required)
	FromEmail string `pulumi:"fromEmail"`

	// FromName is the name that appears in the "From" field (optional)
	FromName *string `pulumi:"fromName,optional"`

	// ReplyTo is the email address for replies (required)
	ReplyTo string `pulumi:"replyTo"`

	// ReplyToName is the name for the reply-to field (optional)
	ReplyToName *string `pulumi:"replyToName,optional"`

	// Address is the street address for the sender (required)
	Address string `pulumi:"address"`

	// Address2 is the second line of the address (optional)
	Address2 *string `pulumi:"address2,optional"`

	// City is the city for the sender address (required)
	City string `pulumi:"city"`

	// State is the state/province for the sender address (optional)
	State *string `pulumi:"state,optional"`

	// Zip is the postal code for the sender address (optional)
	Zip *string `pulumi:"zip,optional"`

	// Country is the country for the sender address (required)
	Country string `pulumi:"country"`
}

// VerifiedSenderState is the state of the VerifiedSender resource.
type VerifiedSenderState struct {
	// Embed the input args in the output state
	VerifiedSenderArgs

	// SenderID is the unique identifier for this verified sender
	SenderID int `pulumi:"senderId"`

	// Verified indicates whether the sender has been verified
	// This is read-only and set by SendGrid after email verification
	Verified bool `pulumi:"verified"`

	// Locked indicates whether the sender is locked
	// This is read-only
	Locked bool `pulumi:"locked"`
}

// Annotate provides descriptions and default values for the VerifiedSender resource.
func (v *VerifiedSender) Annotate(annotator infer.Annotator) {
	annotator.Describe(&v, "Manages a SendGrid Verified Sender.\n\n"+
		"Verified Senders are sender identities that have been verified for sending email. "+
		"After creation, SendGrid will send a verification email to the from_email address. "+
		"The sender must click the verification link to complete the verification process.\n\n"+
		"**Note:** The `verified` status will be `false` until the verification email is confirmed.")
}

// verifiedSenderAPIResponse represents the SendGrid API response structure
type verifiedSenderAPIResponse struct {
	ID          int    `json:"id"`
	Nickname    string `json:"nickname"`
	FromEmail   string `json:"from_email"`
	FromName    string `json:"from_name"`
	ReplyTo     string `json:"reply_to"`
	ReplyToName string `json:"reply_to_name"`
	Address     string `json:"address"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
	Country     string `json:"country"`
	Verified    bool   `json:"verified"`
	Locked      bool   `json:"locked"`
}

// toState converts an API response to VerifiedSenderState
func (r *verifiedSenderAPIResponse) toState() VerifiedSenderState {
	state := VerifiedSenderState{
		VerifiedSenderArgs: VerifiedSenderArgs{
			Nickname:  r.Nickname,
			FromEmail: r.FromEmail,
			ReplyTo:   r.ReplyTo,
			Address:   r.Address,
			City:      r.City,
			Country:   r.Country,
		},
		SenderID: r.ID,
		Verified: r.Verified,
		Locked:   r.Locked,
	}

	// Handle optional fields - only set if non-empty
	if r.FromName != "" {
		state.FromName = &r.FromName
	}
	if r.ReplyToName != "" {
		state.ReplyToName = &r.ReplyToName
	}
	if r.Address2 != "" {
		state.Address2 = &r.Address2
	}
	if r.State != "" {
		state.State = &r.State
	}
	if r.Zip != "" {
		state.Zip = &r.Zip
	}

	return state
}

// Create creates a new SendGrid Verified Sender.
func (v *VerifiedSender) Create(ctx context.Context, req infer.CreateRequest[VerifiedSenderArgs]) (infer.CreateResponse[VerifiedSenderState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := VerifiedSenderState{
			VerifiedSenderArgs: input,
			SenderID:           0,
			Verified:           false,
			Locked:             false,
		}
		return infer.CreateResponse[VerifiedSenderState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[VerifiedSenderState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"nickname":   input.Nickname,
		"from_email": input.FromEmail,
		"reply_to":   input.ReplyTo,
		"address":    input.Address,
		"city":       input.City,
		"country":    input.Country,
	}

	// Add optional fields if provided
	if input.FromName != nil {
		reqBody["from_name"] = *input.FromName
	}
	if input.ReplyToName != nil {
		reqBody["reply_to_name"] = *input.ReplyToName
	}
	if input.Address2 != nil {
		reqBody["address2"] = *input.Address2
	}
	if input.State != nil {
		reqBody["state"] = *input.State
	}
	if input.Zip != nil {
		reqBody["zip"] = *input.Zip
	}

	// Make the API call
	var result verifiedSenderAPIResponse
	if err := client.Post(ctx, "/v3/verified_senders", reqBody, &result); err != nil {
		return infer.CreateResponse[VerifiedSenderState]{}, fmt.Errorf("failed to create verified sender: %w", err)
	}

	state := result.toState()

	return infer.CreateResponse[VerifiedSenderState]{
		ID:     strconv.Itoa(result.ID),
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Verified Sender.
func (v *VerifiedSender) Read(ctx context.Context, req infer.ReadRequest[VerifiedSenderArgs, VerifiedSenderState]) (infer.ReadResponse[VerifiedSenderArgs, VerifiedSenderState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[VerifiedSenderArgs, VerifiedSenderState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// SendGrid doesn't have a GET /verified_senders/{id} endpoint
	// We need to list all and find the one we want
	var listResult struct {
		Results []verifiedSenderAPIResponse `json:"results"`
	}

	if err := client.Get(ctx, "/v3/verified_senders", &listResult); err != nil {
		return infer.ReadResponse[VerifiedSenderArgs, VerifiedSenderState]{}, fmt.Errorf("failed to list verified senders: %w", err)
	}

	// Find the sender by ID
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return infer.ReadResponse[VerifiedSenderArgs, VerifiedSenderState]{}, fmt.Errorf("invalid sender ID: %w", err)
	}

	var found *verifiedSenderAPIResponse
	for i := range listResult.Results {
		if listResult.Results[i].ID == idInt {
			found = &listResult.Results[i]
			break
		}
	}

	if found == nil {
		// Resource no longer exists
		return infer.ReadResponse[VerifiedSenderArgs, VerifiedSenderState]{}, nil
	}

	state := found.toState()
	inputs := state.VerifiedSenderArgs

	return infer.ReadResponse[VerifiedSenderArgs, VerifiedSenderState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Verified Sender.
func (v *VerifiedSender) Update(ctx context.Context, req infer.UpdateRequest[VerifiedSenderArgs, VerifiedSenderState]) (infer.UpdateResponse[VerifiedSenderState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := VerifiedSenderState{
			VerifiedSenderArgs: input,
			SenderID:           oldState.SenderID,
			Verified:           oldState.Verified,
			Locked:             oldState.Locked,
		}
		return infer.UpdateResponse[VerifiedSenderState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[VerifiedSenderState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body with all fields (PATCH requires all fields)
	reqBody := map[string]interface{}{
		"nickname":   input.Nickname,
		"from_email": input.FromEmail,
		"reply_to":   input.ReplyTo,
		"address":    input.Address,
		"city":       input.City,
		"country":    input.Country,
	}

	// Add optional fields - use empty string if nil
	if input.FromName != nil {
		reqBody["from_name"] = *input.FromName
	} else {
		reqBody["from_name"] = ""
	}
	if input.ReplyToName != nil {
		reqBody["reply_to_name"] = *input.ReplyToName
	} else {
		reqBody["reply_to_name"] = ""
	}
	if input.Address2 != nil {
		reqBody["address2"] = *input.Address2
	} else {
		reqBody["address2"] = ""
	}
	if input.State != nil {
		reqBody["state"] = *input.State
	} else {
		reqBody["state"] = ""
	}
	if input.Zip != nil {
		reqBody["zip"] = *input.Zip
	} else {
		reqBody["zip"] = ""
	}

	var result verifiedSenderAPIResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/verified_senders/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[VerifiedSenderState]{}, fmt.Errorf("failed to update verified sender: %w", err)
	}

	state := result.toState()

	return infer.UpdateResponse[VerifiedSenderState]{Output: state}, nil
}

// Delete removes a SendGrid Verified Sender.
func (v *VerifiedSender) Delete(ctx context.Context, req infer.DeleteRequest[VerifiedSenderState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/verified_senders/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete verified sender: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
