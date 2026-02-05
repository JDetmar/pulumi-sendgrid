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

// Template is the controller for the SendGrid Template resource.
//
// This resource manages SendGrid Transactional Templates, which are used to
// create and manage email templates for transactional emails.
type Template struct{}

// TemplateGeneration represents the generation type for a template.
// "legacy" templates support plain text and HTML content.
// "dynamic" templates support handlebars syntax for dynamic content.
type TemplateGeneration string

const (
	// TemplateGenerationLegacy is for legacy templates (plain text/HTML)
	TemplateGenerationLegacy TemplateGeneration = "legacy"
	// TemplateGenerationDynamic is for dynamic templates (handlebars syntax)
	TemplateGenerationDynamic TemplateGeneration = "dynamic"
)

// TemplateArgs are the inputs to the Template resource.
type TemplateArgs struct {
	// Name is the name of the template (required, max 100 characters)
	Name string `pulumi:"name"`

	// Generation is the type of template: "legacy" or "dynamic" (required)
	// - "legacy": Supports plain text and HTML content
	// - "dynamic": Supports handlebars syntax for dynamic content
	// Once set, this cannot be changed.
	Generation TemplateGeneration `pulumi:"generation"`
}

// TemplateVersionSummary represents a summary of a template version (read-only).
// Full version management is done via the TemplateVersion resource.
type TemplateVersionSummary struct {
	// ID is the unique identifier for the template version
	ID string `pulumi:"id"`

	// TemplateID is the ID of the parent template
	TemplateID string `pulumi:"templateId"`

	// Name is the name of the version
	Name string `pulumi:"name"`

	// Active indicates if this version is the active version
	Active bool `pulumi:"active"`

	// UpdatedAt is the timestamp when this version was last updated
	UpdatedAt string `pulumi:"updatedAt,optional"`
}

// TemplateState is the state of the Template resource.
type TemplateState struct {
	// Embed the input args in the output state
	TemplateArgs

	// TemplateID is the unique identifier for this template
	TemplateID string `pulumi:"templateId"`

	// UpdatedAt is the timestamp when the template was last updated
	UpdatedAt string `pulumi:"updatedAt,optional"`

	// Versions is the list of template versions (read-only)
	// Use the TemplateVersion resource to manage versions
	Versions []TemplateVersionSummary `pulumi:"versions,optional"`
}

// Annotate provides descriptions and default values for the Template resource.
func (t *Template) Annotate(annotator infer.Annotator) {
	annotator.Describe(&t, "Manages a SendGrid Transactional Template.\n\n"+
		"Transactional templates are used to create reusable email templates "+
		"for transactional emails like receipts, password resets, etc.\n\n"+
		"Templates can be either 'legacy' (plain text/HTML) or 'dynamic' "+
		"(supporting handlebars syntax for personalization).\n\n"+
		"**Note:** Template versions are managed separately via the TemplateVersion resource.")
}

// Create creates a new SendGrid Template.
func (t *Template) Create(ctx context.Context, req infer.CreateRequest[TemplateArgs]) (infer.CreateResponse[TemplateState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := TemplateState{
			TemplateArgs: input,
			TemplateID:   "[computed]",
			UpdatedAt:    "[computed]",
			Versions:     []TemplateVersionSummary{},
		}
		return infer.CreateResponse[TemplateState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[TemplateState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"name":       input.Name,
		"generation": string(input.Generation),
	}

	// Make the API call
	var result struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Generation string `json:"generation"`
		UpdatedAt  string `json:"updated_at"`
		Versions   []struct {
			ID         string `json:"id"`
			TemplateID string `json:"template_id"`
			Name       string `json:"name"`
			Active     int    `json:"active"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"versions"`
	}

	if err := client.Post(ctx, "/v3/templates", reqBody, &result); err != nil {
		return infer.CreateResponse[TemplateState]{}, fmt.Errorf("failed to create template: %w", err)
	}

	// Convert versions to summary format
	versions := make([]TemplateVersionSummary, len(result.Versions))
	for i, v := range result.Versions {
		versions[i] = TemplateVersionSummary{
			ID:         v.ID,
			TemplateID: v.TemplateID,
			Name:       v.Name,
			Active:     v.Active == 1,
			UpdatedAt:  v.UpdatedAt,
		}
	}

	state := TemplateState{
		TemplateArgs: TemplateArgs{
			Name:       result.Name,
			Generation: TemplateGeneration(result.Generation),
		},
		TemplateID: result.ID,
		UpdatedAt:  result.UpdatedAt,
		Versions:   versions,
	}

	return infer.CreateResponse[TemplateState]{
		ID:     result.ID,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Template.
func (t *Template) Read(ctx context.Context, req infer.ReadRequest[TemplateArgs, TemplateState]) (infer.ReadResponse[TemplateArgs, TemplateState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[TemplateArgs, TemplateState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the template details
	var result struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Generation string `json:"generation"`
		UpdatedAt  string `json:"updated_at"`
		Versions   []struct {
			ID         string `json:"id"`
			TemplateID string `json:"template_id"`
			Name       string `json:"name"`
			Active     int    `json:"active"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"versions"`
	}

	if err := client.Get(ctx, fmt.Sprintf("/v3/templates/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[TemplateArgs, TemplateState]{}, nil
		}
		return infer.ReadResponse[TemplateArgs, TemplateState]{}, fmt.Errorf("failed to read template: %w", err)
	}

	// Convert versions to summary format
	versions := make([]TemplateVersionSummary, len(result.Versions))
	for i, v := range result.Versions {
		versions[i] = TemplateVersionSummary{
			ID:         v.ID,
			TemplateID: v.TemplateID,
			Name:       v.Name,
			Active:     v.Active == 1,
			UpdatedAt:  v.UpdatedAt,
		}
	}

	// Update state with values from API
	state := TemplateState{
		TemplateArgs: TemplateArgs{
			Name:       result.Name,
			Generation: TemplateGeneration(result.Generation),
		},
		TemplateID: result.ID,
		UpdatedAt:  result.UpdatedAt,
		Versions:   versions,
	}

	inputs := TemplateArgs{
		Name:       result.Name,
		Generation: TemplateGeneration(result.Generation),
	}

	return infer.ReadResponse[TemplateArgs, TemplateState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Template.
func (t *Template) Update(ctx context.Context, req infer.UpdateRequest[TemplateArgs, TemplateState]) (infer.UpdateResponse[TemplateState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := TemplateState{
			TemplateArgs: input,
			TemplateID:   oldState.TemplateID,
			UpdatedAt:    oldState.UpdatedAt,
			Versions:     oldState.Versions,
		}
		return infer.UpdateResponse[TemplateState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[TemplateState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Note: SendGrid only allows updating the name via PATCH
	// Generation cannot be changed after creation
	reqBody := map[string]interface{}{
		"name": input.Name,
	}

	var result struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Generation string `json:"generation"`
		UpdatedAt  string `json:"updated_at"`
		Versions   []struct {
			ID         string `json:"id"`
			TemplateID string `json:"template_id"`
			Name       string `json:"name"`
			Active     int    `json:"active"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"versions"`
	}

	if err := client.Patch(ctx, fmt.Sprintf("/v3/templates/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[TemplateState]{}, fmt.Errorf("failed to update template: %w", err)
	}

	// Convert versions to summary format
	versions := make([]TemplateVersionSummary, len(result.Versions))
	for i, v := range result.Versions {
		versions[i] = TemplateVersionSummary{
			ID:         v.ID,
			TemplateID: v.TemplateID,
			Name:       v.Name,
			Active:     v.Active == 1,
			UpdatedAt:  v.UpdatedAt,
		}
	}

	state := TemplateState{
		TemplateArgs: TemplateArgs{
			Name:       result.Name,
			Generation: TemplateGeneration(result.Generation),
		},
		TemplateID: result.ID,
		UpdatedAt:  result.UpdatedAt,
		Versions:   versions,
	}

	return infer.UpdateResponse[TemplateState]{Output: state}, nil
}

// Delete removes a SendGrid Template.
func (t *Template) Delete(ctx context.Context, req infer.DeleteRequest[TemplateState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/templates/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete template: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
