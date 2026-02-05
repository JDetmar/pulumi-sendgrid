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

// Alert is the controller for the SendGrid Alert resource.
//
// This resource manages SendGrid Alerts, which notify you about usage limits
// and statistics via email.
type Alert struct{}

// AlertArgs are the inputs to the Alert resource.
type AlertArgs struct {
	// Type is the type of alert (required)
	// Valid values: "usage_limit" or "stats_notification"
	Type string `pulumi:"type"`

	// EmailTo is the email address to send alerts to (required)
	EmailTo string `pulumi:"emailTo"`

	// Percentage is the usage threshold for usage_limit alerts (required for usage_limit)
	// Alerts are triggered when this percentage of your plan's email limit is reached
	Percentage *int `pulumi:"percentage,optional"`

	// Frequency is how often to send stats_notification alerts (required for stats_notification)
	// Valid values: "daily", "weekly", or "monthly"
	Frequency *string `pulumi:"frequency,optional"`
}

// AlertState is the state of the Alert resource.
type AlertState struct {
	// Embed the input args in the output state
	AlertArgs

	// AlertID is the unique identifier assigned by SendGrid
	AlertID int `pulumi:"alertId"`

	// CreatedAt is the Unix timestamp when the alert was created
	CreatedAt int64 `pulumi:"createdAt"`

	// UpdatedAt is the Unix timestamp when the alert was last updated
	UpdatedAt int64 `pulumi:"updatedAt"`
}

// Annotate provides descriptions for the Alert resource.
func (a *Alert) Annotate(annotator infer.Annotator) {
	annotator.Describe(&a, "Manages a SendGrid Alert.\n\n"+
		"Alerts notify you via email about important account events. Two types are available:\n\n"+
		"1. **usage_limit**: Notifies when your email usage reaches a specified percentage of your plan limit.\n"+
		"2. **stats_notification**: Sends periodic email statistics (daily, weekly, or monthly).\n\n"+
		"You can create multiple alerts of the same type with different email recipients.")
}

// alertAPIResponse represents the SendGrid API response for alerts
type alertAPIResponse struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	EmailTo    string `json:"email_to"`
	Percentage int    `json:"percentage,omitempty"`
	Frequency  string `json:"frequency,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// toState converts an API response to AlertState
func (r *alertAPIResponse) toState() AlertState {
	var percentage *int
	if r.Percentage > 0 {
		percentage = &r.Percentage
	}

	var frequency *string
	if r.Frequency != "" {
		frequency = &r.Frequency
	}

	return AlertState{
		AlertArgs: AlertArgs{
			Type:       r.Type,
			EmailTo:    r.EmailTo,
			Percentage: percentage,
			Frequency:  frequency,
		},
		AlertID:   r.ID,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// Create creates a new SendGrid Alert.
func (a *Alert) Create(ctx context.Context, req infer.CreateRequest[AlertArgs]) (infer.CreateResponse[AlertState], error) {
	input := req.Inputs
	preview := req.DryRun

	// Validate inputs based on alert type
	if input.Type == "usage_limit" && input.Percentage == nil {
		return infer.CreateResponse[AlertState]{}, fmt.Errorf("percentage is required for usage_limit alerts")
	}
	if input.Type == "stats_notification" && input.Frequency == nil {
		return infer.CreateResponse[AlertState]{}, fmt.Errorf("frequency is required for stats_notification alerts")
	}

	// During preview, return placeholder state
	if preview {
		state := AlertState{
			AlertArgs: input,
			AlertID:   0,
			CreatedAt: 0,
			UpdatedAt: 0,
		}
		return infer.CreateResponse[AlertState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[AlertState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"type":     input.Type,
		"email_to": input.EmailTo,
	}
	if input.Percentage != nil {
		reqBody["percentage"] = *input.Percentage
	}
	if input.Frequency != nil {
		reqBody["frequency"] = *input.Frequency
	}

	// Make the API call to create alert
	// POST /v3/alerts
	var result alertAPIResponse
	if err := client.Post(ctx, "/v3/alerts", reqBody, &result); err != nil {
		return infer.CreateResponse[AlertState]{}, fmt.Errorf("failed to create alert: %w", err)
	}

	state := result.toState()

	return infer.CreateResponse[AlertState]{
		ID:     strconv.Itoa(result.ID),
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Alert.
func (a *Alert) Read(ctx context.Context, req infer.ReadRequest[AlertArgs, AlertState]) (infer.ReadResponse[AlertArgs, AlertState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[AlertArgs, AlertState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Get alert details
	// GET /v3/alerts/{alert_id}
	var result alertAPIResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/alerts/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.ReadResponse[AlertArgs, AlertState]{}, nil
		}
		return infer.ReadResponse[AlertArgs, AlertState]{}, fmt.Errorf("failed to read alert: %w", err)
	}

	state := result.toState()
	inputs := state.AlertArgs

	return infer.ReadResponse[AlertArgs, AlertState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Alert.
func (a *Alert) Update(ctx context.Context, req infer.UpdateRequest[AlertArgs, AlertState]) (infer.UpdateResponse[AlertState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// Validate inputs based on alert type
	if input.Type == "usage_limit" && input.Percentage == nil {
		return infer.UpdateResponse[AlertState]{}, fmt.Errorf("percentage is required for usage_limit alerts")
	}
	if input.Type == "stats_notification" && input.Frequency == nil {
		return infer.UpdateResponse[AlertState]{}, fmt.Errorf("frequency is required for stats_notification alerts")
	}

	// During preview, return expected state
	if preview {
		state := AlertState{
			AlertArgs: input,
			AlertID:   oldState.AlertID,
			CreatedAt: oldState.CreatedAt,
			UpdatedAt: oldState.UpdatedAt,
		}
		return infer.UpdateResponse[AlertState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[AlertState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"email_to": input.EmailTo,
	}
	if input.Percentage != nil {
		reqBody["percentage"] = *input.Percentage
	}
	if input.Frequency != nil {
		reqBody["frequency"] = *input.Frequency
	}

	// Make the API call (PATCH to update the alert)
	// PATCH /v3/alerts/{alert_id}
	var result alertAPIResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/alerts/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[AlertState]{}, fmt.Errorf("failed to update alert: %w", err)
	}

	state := result.toState()

	return infer.UpdateResponse[AlertState]{Output: state}, nil
}

// Delete removes a SendGrid Alert.
func (a *Alert) Delete(ctx context.Context, req infer.DeleteRequest[AlertState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Delete the alert
	// DELETE /v3/alerts/{alert_id}
	if err := client.Delete(ctx, fmt.Sprintf("/v3/alerts/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete alert: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
