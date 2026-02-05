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

// TemplateVersion is the controller for the SendGrid Template Version resource.
//
// This resource manages versions of SendGrid Transactional Templates.
// Template versions contain the actual content (subject, HTML, plain text) of emails.
type TemplateVersion struct{}

// TemplateVersionEditor represents the editor type used to create the template version.
type TemplateVersionEditor string

const (
	// TemplateVersionEditorCode is for code-based template editing
	TemplateVersionEditorCode TemplateVersionEditor = "code"
	// TemplateVersionEditorDesign is for design-based template editing (Design Editor)
	TemplateVersionEditorDesign TemplateVersionEditor = "design"
)

// TemplateVersionArgs are the inputs to the TemplateVersion resource.
type TemplateVersionArgs struct {
	// TemplateID is the ID of the parent template (required)
	TemplateID string `pulumi:"templateId"`

	// Name is the name of the template version (required)
	Name string `pulumi:"name"`

	// Subject is the subject line of the email (required for dynamic templates)
	Subject *string `pulumi:"subject,optional"`

	// HtmlContent is the HTML content of the email
	HtmlContent *string `pulumi:"htmlContent,optional"`

	// PlainContent is the plain text content of the email
	PlainContent *string `pulumi:"plainContent,optional"`

	// Active indicates if this version should be the active version (0 or 1)
	// Only one version can be active at a time
	Active *int `pulumi:"active,optional"`

	// Editor is the editor type used: "code" or "design"
	Editor *TemplateVersionEditor `pulumi:"editor,optional"`

	// GeneratePlainContent indicates whether to auto-generate plain text from HTML
	GeneratePlainContent *bool `pulumi:"generatePlainContent,optional"`

	// TestData is JSON data that can be used in template testing/preview
	TestData *string `pulumi:"testData,optional"`
}

// TemplateVersionState is the state of the TemplateVersion resource.
type TemplateVersionState struct {
	// Embed the input args in the output state
	TemplateVersionArgs

	// VersionID is the unique identifier for this template version
	VersionID string `pulumi:"versionId"`

	// UpdatedAt is the timestamp when the version was last updated
	UpdatedAt string `pulumi:"updatedAt,optional"`

	// ThumbnailURL is the URL of the thumbnail for the template version
	ThumbnailURL string `pulumi:"thumbnailUrl,optional"`
}

// Annotate provides descriptions and default values for the TemplateVersion resource.
func (tv *TemplateVersion) Annotate(annotator infer.Annotator) {
	annotator.Describe(&tv, "Manages a SendGrid Template Version.\n\n"+
		"Template versions contain the actual content of transactional emails, "+
		"including the subject line, HTML content, and plain text content.\n\n"+
		"Each template can have multiple versions, but only one can be active at a time. "+
		"The active version is used when sending emails through the template.\n\n"+
		"**Note:** Dynamic templates support handlebars syntax for personalization.")
}

// Create creates a new SendGrid Template Version.
func (tv *TemplateVersion) Create(ctx context.Context, req infer.CreateRequest[TemplateVersionArgs]) (infer.CreateResponse[TemplateVersionState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := TemplateVersionState{
			TemplateVersionArgs: input,
			VersionID:           "[computed]",
			UpdatedAt:           "[computed]",
		}
		return infer.CreateResponse[TemplateVersionState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[TemplateVersionState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"name": input.Name,
	}
	if input.Subject != nil {
		reqBody["subject"] = *input.Subject
	}
	if input.HtmlContent != nil {
		reqBody["html_content"] = *input.HtmlContent
	}
	if input.PlainContent != nil {
		reqBody["plain_content"] = *input.PlainContent
	}
	if input.Active != nil {
		reqBody["active"] = *input.Active
	}
	if input.Editor != nil {
		reqBody["editor"] = string(*input.Editor)
	}
	if input.GeneratePlainContent != nil {
		reqBody["generate_plain_content"] = *input.GeneratePlainContent
	}
	if input.TestData != nil {
		reqBody["test_data"] = *input.TestData
	}

	// Make the API call
	var result struct {
		ID                   string `json:"id"`
		TemplateID           string `json:"template_id"`
		Name                 string `json:"name"`
		Subject              string `json:"subject"`
		HtmlContent          string `json:"html_content"`
		PlainContent         string `json:"plain_content"`
		Active               int    `json:"active"`
		Editor               string `json:"editor"`
		GeneratePlainContent bool   `json:"generate_plain_content"`
		TestData             string `json:"test_data"`
		UpdatedAt            string `json:"updated_at"`
		ThumbnailURL         string `json:"thumbnail_url"`
	}

	path := fmt.Sprintf("/v3/templates/%s/versions", input.TemplateID)
	if err := client.Post(ctx, path, reqBody, &result); err != nil {
		return infer.CreateResponse[TemplateVersionState]{}, fmt.Errorf("failed to create template version: %w", err)
	}

	// Convert result to state
	state := buildTemplateVersionState(result)

	return infer.CreateResponse[TemplateVersionState]{
		ID:     result.ID,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Template Version.
func (tv *TemplateVersion) Read(ctx context.Context, req infer.ReadRequest[TemplateVersionArgs, TemplateVersionState]) (infer.ReadResponse[TemplateVersionArgs, TemplateVersionState], error) {
	id := req.ID
	oldState := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[TemplateVersionArgs, TemplateVersionState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the template version details
	var result struct {
		ID                   string `json:"id"`
		TemplateID           string `json:"template_id"`
		Name                 string `json:"name"`
		Subject              string `json:"subject"`
		HtmlContent          string `json:"html_content"`
		PlainContent         string `json:"plain_content"`
		Active               int    `json:"active"`
		Editor               string `json:"editor"`
		GeneratePlainContent bool   `json:"generate_plain_content"`
		TestData             string `json:"test_data"`
		UpdatedAt            string `json:"updated_at"`
		ThumbnailURL         string `json:"thumbnail_url"`
	}

	// Use the template ID from old state since it's required for the path
	templateID := oldState.TemplateID
	path := fmt.Sprintf("/v3/templates/%s/versions/%s", templateID, id)
	if err := client.Get(ctx, path, &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[TemplateVersionArgs, TemplateVersionState]{}, nil
		}
		return infer.ReadResponse[TemplateVersionArgs, TemplateVersionState]{}, fmt.Errorf("failed to read template version: %w", err)
	}

	// Convert result to state
	state := buildTemplateVersionState(result)

	// Build inputs from state
	inputs := TemplateVersionArgs{
		TemplateID:           state.TemplateID,
		Name:                 state.Name,
		Subject:              state.Subject,
		HtmlContent:          state.HtmlContent,
		PlainContent:         state.PlainContent,
		Active:               state.Active,
		Editor:               state.Editor,
		GeneratePlainContent: state.GeneratePlainContent,
		TestData:             state.TestData,
	}

	return infer.ReadResponse[TemplateVersionArgs, TemplateVersionState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Template Version.
func (tv *TemplateVersion) Update(ctx context.Context, req infer.UpdateRequest[TemplateVersionArgs, TemplateVersionState]) (infer.UpdateResponse[TemplateVersionState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := TemplateVersionState{
			TemplateVersionArgs: input,
			VersionID:           oldState.VersionID,
			UpdatedAt:           oldState.UpdatedAt,
			ThumbnailURL:        oldState.ThumbnailURL,
		}
		return infer.UpdateResponse[TemplateVersionState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[TemplateVersionState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"name": input.Name,
	}
	if input.Subject != nil {
		reqBody["subject"] = *input.Subject
	}
	if input.HtmlContent != nil {
		reqBody["html_content"] = *input.HtmlContent
	}
	if input.PlainContent != nil {
		reqBody["plain_content"] = *input.PlainContent
	}
	if input.Active != nil {
		reqBody["active"] = *input.Active
	}
	if input.GeneratePlainContent != nil {
		reqBody["generate_plain_content"] = *input.GeneratePlainContent
	}
	if input.TestData != nil {
		reqBody["test_data"] = *input.TestData
	}

	var result struct {
		ID                   string `json:"id"`
		TemplateID           string `json:"template_id"`
		Name                 string `json:"name"`
		Subject              string `json:"subject"`
		HtmlContent          string `json:"html_content"`
		PlainContent         string `json:"plain_content"`
		Active               int    `json:"active"`
		Editor               string `json:"editor"`
		GeneratePlainContent bool   `json:"generate_plain_content"`
		TestData             string `json:"test_data"`
		UpdatedAt            string `json:"updated_at"`
		ThumbnailURL         string `json:"thumbnail_url"`
	}

	path := fmt.Sprintf("/v3/templates/%s/versions/%s", input.TemplateID, id)
	if err := client.Patch(ctx, path, reqBody, &result); err != nil {
		return infer.UpdateResponse[TemplateVersionState]{}, fmt.Errorf("failed to update template version: %w", err)
	}

	// Convert result to state
	state := buildTemplateVersionState(result)

	return infer.UpdateResponse[TemplateVersionState]{Output: state}, nil
}

// Delete removes a SendGrid Template Version.
func (tv *TemplateVersion) Delete(ctx context.Context, req infer.DeleteRequest[TemplateVersionState]) (infer.DeleteResponse, error) {
	id := req.ID
	state := req.State

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	path := fmt.Sprintf("/v3/templates/%s/versions/%s", state.TemplateID, id)
	if err := client.Delete(ctx, path); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete template version: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

// buildTemplateVersionState converts API response to TemplateVersionState
func buildTemplateVersionState(result struct {
	ID                   string `json:"id"`
	TemplateID           string `json:"template_id"`
	Name                 string `json:"name"`
	Subject              string `json:"subject"`
	HtmlContent          string `json:"html_content"`
	PlainContent         string `json:"plain_content"`
	Active               int    `json:"active"`
	Editor               string `json:"editor"`
	GeneratePlainContent bool   `json:"generate_plain_content"`
	TestData             string `json:"test_data"`
	UpdatedAt            string `json:"updated_at"`
	ThumbnailURL         string `json:"thumbnail_url"`
}) TemplateVersionState {
	// Convert optional string fields
	var subject, htmlContent, plainContent, testData *string
	if result.Subject != "" {
		subject = &result.Subject
	}
	if result.HtmlContent != "" {
		htmlContent = &result.HtmlContent
	}
	if result.PlainContent != "" {
		plainContent = &result.PlainContent
	}
	if result.TestData != "" {
		testData = &result.TestData
	}

	// Convert active field
	var active *int
	activeVal := result.Active
	active = &activeVal

	// Convert editor field
	var editor *TemplateVersionEditor
	if result.Editor != "" {
		e := TemplateVersionEditor(result.Editor)
		editor = &e
	}

	// Convert generatePlainContent
	var generatePlainContent = &result.GeneratePlainContent

	return TemplateVersionState{
		TemplateVersionArgs: TemplateVersionArgs{
			TemplateID:           result.TemplateID,
			Name:                 result.Name,
			Subject:              subject,
			HtmlContent:          htmlContent,
			PlainContent:         plainContent,
			Active:               active,
			Editor:               editor,
			GeneratePlainContent: generatePlainContent,
			TestData:             testData,
		},
		VersionID:    result.ID,
		UpdatedAt:    result.UpdatedAt,
		ThumbnailURL: result.ThumbnailURL,
	}
}
