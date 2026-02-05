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

// EventWebhook is the controller for the SendGrid Event Webhook resource.
//
// This resource manages SendGrid Event Webhooks, which allow you to receive
// notifications about email events (delivered, opened, clicked, bounced, etc.)
// at a specified URL endpoint.
type EventWebhook struct{}

// EventWebhookArgs are the inputs to the EventWebhook resource.
type EventWebhookArgs struct {
	// URL is the endpoint where SendGrid will POST event data (required)
	URL string `pulumi:"url"`

	// Enabled indicates whether the webhook is active (optional, defaults to true)
	Enabled *bool `pulumi:"enabled,optional"`

	// FriendlyName is a human-readable name for the webhook (optional)
	FriendlyName *string `pulumi:"friendlyName,optional"`

	// Event types to subscribe to (all optional, default to false)

	// Bounce - message was bounced
	Bounce *bool `pulumi:"bounce,optional"`

	// Click - recipient clicked a link
	Click *bool `pulumi:"click,optional"`

	// Deferred - message delivery was deferred
	Deferred *bool `pulumi:"deferred,optional"`

	// Delivered - message was delivered
	Delivered *bool `pulumi:"delivered,optional"`

	// Dropped - message was dropped
	Dropped *bool `pulumi:"dropped,optional"`

	// Open - recipient opened the message
	Open *bool `pulumi:"open,optional"`

	// Processed - message was processed
	Processed *bool `pulumi:"processed,optional"`

	// SpamReport - recipient marked as spam
	SpamReport *bool `pulumi:"spamReport,optional"`

	// Unsubscribe - recipient unsubscribed
	Unsubscribe *bool `pulumi:"unsubscribe,optional"`

	// GroupResubscribe - recipient resubscribed to a group
	GroupResubscribe *bool `pulumi:"groupResubscribe,optional"`

	// GroupUnsubscribe - recipient unsubscribed from a group
	GroupUnsubscribe *bool `pulumi:"groupUnsubscribe,optional"`
}

// EventWebhookState is the state of the EventWebhook resource.
type EventWebhookState struct {
	// Embed the input args in the output state
	EventWebhookArgs

	// WebhookID is the unique identifier assigned by SendGrid
	WebhookID string `pulumi:"webhookId"`
}

// Annotate provides descriptions for the EventWebhook resource.
func (w *EventWebhook) Annotate(annotator infer.Annotator) {
	annotator.Describe(&w, "Manages a SendGrid Event Webhook.\n\n"+
		"Event Webhooks allow you to receive HTTP POST notifications when email events "+
		"occur, such as delivery, opens, clicks, bounces, and more. Configure the URL "+
		"endpoint and select which events to track.\n\n"+
		"Note: Only one webhook can be configured per URL. Signature verification "+
		"must be configured separately after webhook creation.")
}

// eventWebhookAPIResponse represents the SendGrid API response structure for event webhooks
type eventWebhookAPIResponse struct {
	ID               string `json:"id"`
	URL              string `json:"url"`
	Enabled          bool   `json:"enabled"`
	FriendlyName     string `json:"friendly_name,omitempty"`
	Bounce           bool   `json:"bounce"`
	Click            bool   `json:"click"`
	Deferred         bool   `json:"deferred"`
	Delivered        bool   `json:"delivered"`
	Dropped          bool   `json:"dropped"`
	Open             bool   `json:"open"`
	Processed        bool   `json:"processed"`
	SpamReport       bool   `json:"spam_report"`
	Unsubscribe      bool   `json:"unsubscribe"`
	GroupResubscribe bool   `json:"group_resubscribe"`
	GroupUnsubscribe bool   `json:"group_unsubscribe"`
}

// toState converts an API response to EventWebhookState
func (r *eventWebhookAPIResponse) toState() EventWebhookState {
	var friendlyName *string
	if r.FriendlyName != "" {
		friendlyName = &r.FriendlyName
	}

	enabled := r.Enabled
	bounce := r.Bounce
	click := r.Click
	deferred := r.Deferred
	delivered := r.Delivered
	dropped := r.Dropped
	open := r.Open
	processed := r.Processed
	spamReport := r.SpamReport
	unsubscribe := r.Unsubscribe
	groupResubscribe := r.GroupResubscribe
	groupUnsubscribe := r.GroupUnsubscribe

	return EventWebhookState{
		EventWebhookArgs: EventWebhookArgs{
			URL:              r.URL,
			Enabled:          &enabled,
			FriendlyName:     friendlyName,
			Bounce:           &bounce,
			Click:            &click,
			Deferred:         &deferred,
			Delivered:        &delivered,
			Dropped:          &dropped,
			Open:             &open,
			Processed:        &processed,
			SpamReport:       &spamReport,
			Unsubscribe:      &unsubscribe,
			GroupResubscribe: &groupResubscribe,
			GroupUnsubscribe: &groupUnsubscribe,
		},
		WebhookID: r.ID,
	}
}

// buildRequestBody creates the API request body from EventWebhookArgs
func (args *EventWebhookArgs) buildRequestBody() map[string]interface{} {
	reqBody := map[string]interface{}{
		"url": args.URL,
	}

	if args.Enabled != nil {
		reqBody["enabled"] = *args.Enabled
	} else {
		reqBody["enabled"] = true // default to enabled
	}

	if args.FriendlyName != nil {
		reqBody["friendly_name"] = *args.FriendlyName
	}

	// Event types - default to false if not specified
	if args.Bounce != nil {
		reqBody["bounce"] = *args.Bounce
	}
	if args.Click != nil {
		reqBody["click"] = *args.Click
	}
	if args.Deferred != nil {
		reqBody["deferred"] = *args.Deferred
	}
	if args.Delivered != nil {
		reqBody["delivered"] = *args.Delivered
	}
	if args.Dropped != nil {
		reqBody["dropped"] = *args.Dropped
	}
	if args.Open != nil {
		reqBody["open"] = *args.Open
	}
	if args.Processed != nil {
		reqBody["processed"] = *args.Processed
	}
	if args.SpamReport != nil {
		reqBody["spam_report"] = *args.SpamReport
	}
	if args.Unsubscribe != nil {
		reqBody["unsubscribe"] = *args.Unsubscribe
	}
	if args.GroupResubscribe != nil {
		reqBody["group_resubscribe"] = *args.GroupResubscribe
	}
	if args.GroupUnsubscribe != nil {
		reqBody["group_unsubscribe"] = *args.GroupUnsubscribe
	}

	return reqBody
}

// Create creates a new SendGrid Event Webhook.
func (w *EventWebhook) Create(ctx context.Context, req infer.CreateRequest[EventWebhookArgs]) (infer.CreateResponse[EventWebhookState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		state := EventWebhookState{
			EventWebhookArgs: input,
			WebhookID:        "[computed]",
		}
		if state.Enabled == nil {
			state.Enabled = &enabled
		}
		return infer.CreateResponse[EventWebhookState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[EventWebhookState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := input.buildRequestBody()

	// Make the API call
	// POST /v3/user/webhooks/event/settings
	var result eventWebhookAPIResponse
	if err := client.Post(ctx, "/v3/user/webhooks/event/settings", reqBody, &result); err != nil {
		return infer.CreateResponse[EventWebhookState]{}, fmt.Errorf("failed to create event webhook: %w", err)
	}

	state := result.toState()

	return infer.CreateResponse[EventWebhookState]{
		ID:     result.ID,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Event Webhook.
func (w *EventWebhook) Read(ctx context.Context, req infer.ReadRequest[EventWebhookArgs, EventWebhookState]) (infer.ReadResponse[EventWebhookArgs, EventWebhookState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[EventWebhookArgs, EventWebhookState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the webhook details
	// GET /v3/user/webhooks/event/settings/{id}
	var result eventWebhookAPIResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/user/webhooks/event/settings/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[EventWebhookArgs, EventWebhookState]{}, nil
		}
		return infer.ReadResponse[EventWebhookArgs, EventWebhookState]{}, fmt.Errorf("failed to read event webhook: %w", err)
	}

	state := result.toState()
	inputs := state.EventWebhookArgs

	return infer.ReadResponse[EventWebhookArgs, EventWebhookState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Event Webhook.
func (w *EventWebhook) Update(ctx context.Context, req infer.UpdateRequest[EventWebhookArgs, EventWebhookState]) (infer.UpdateResponse[EventWebhookState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := EventWebhookState{
			EventWebhookArgs: input,
			WebhookID:        oldState.WebhookID,
		}
		return infer.UpdateResponse[EventWebhookState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[EventWebhookState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body
	reqBody := input.buildRequestBody()

	// Make the API call (PATCH to update the webhook)
	// PATCH /v3/user/webhooks/event/settings/{id}
	var result eventWebhookAPIResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/user/webhooks/event/settings/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[EventWebhookState]{}, fmt.Errorf("failed to update event webhook: %w", err)
	}

	state := result.toState()

	return infer.UpdateResponse[EventWebhookState]{Output: state}, nil
}

// Delete removes a SendGrid Event Webhook.
func (w *EventWebhook) Delete(ctx context.Context, req infer.DeleteRequest[EventWebhookState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	// DELETE /v3/user/webhooks/event/settings/{id}
	if err := client.Delete(ctx, fmt.Sprintf("/v3/user/webhooks/event/settings/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete event webhook: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
