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

// LinkBranding is the controller for the SendGrid Link Branding resource.
//
// This resource manages SendGrid Link Branding (formerly "Link Whitelabel"),
// which allows you to brand the links in your emails with your own domain
// instead of the default SendGrid domain.
type LinkBranding struct{}

// LinkBrandingArgs are the inputs to the LinkBranding resource.
type LinkBrandingArgs struct {
	// Domain is the root domain for the subdomain being used to brand links (required)
	Domain string `pulumi:"domain"`

	// Subdomain is the subdomain to use for branded links (optional)
	// If not provided, SendGrid will generate one.
	Subdomain *string `pulumi:"subdomain,optional"`

	// Default marks this link branding as the default for the domain (optional)
	Default *bool `pulumi:"default,optional"`

	// Region is the region for the link branding: "global" or "eu" (optional, default: global)
	Region *string `pulumi:"region,optional"`
}

// LinkBrandingDNSRecord represents a DNS record required for link branding
type LinkBrandingDNSRecord struct {
	// Valid indicates if the record has been validated
	Valid bool `pulumi:"valid"`
	// Type is the DNS record type (typically CNAME)
	Type string `pulumi:"type"`
	// Host is the hostname for the record
	Host string `pulumi:"host"`
	// Data is the value/data for the record
	Data string `pulumi:"data"`
}

// LinkBrandingState is the state of the LinkBranding resource.
type LinkBrandingState struct {
	// Embed the input args in the output state
	LinkBrandingArgs

	// LinkID is the unique identifier for this link branding
	LinkID int `pulumi:"linkId"`

	// UserID is the ID of the user that this link branding is associated with
	UserID int `pulumi:"userId"`

	// Username is the username associated with this link branding
	Username string `pulumi:"username"`

	// Valid indicates whether the link branding has been validated
	Valid bool `pulumi:"valid"`

	// Legacy indicates if this is a legacy whitelabel
	Legacy bool `pulumi:"legacy"`

	// OwnerCname is the CNAME record for the owner verification
	OwnerCname *LinkBrandingDNSRecord `pulumi:"ownerCname,optional"`

	// BrandCname is the CNAME record for branding
	BrandCname *LinkBrandingDNSRecord `pulumi:"brandCname,optional"`
}

// Annotate provides descriptions for the LinkBranding resource.
func (l *LinkBranding) Annotate(annotator infer.Annotator) {
	annotator.Describe(&l, "Manages a SendGrid Link Branding.\n\n"+
		"Link Branding (formerly Link Whitelabel) allows you to customize the links in your emails "+
		"to use your own domain instead of sendgrid.net. This helps improve deliverability and "+
		"brand recognition.\n\n"+
		"After creating this resource, you must add the DNS records to your domain's DNS settings "+
		"and then validate the link branding using the SendGrid console or API.")
}

// linkBrandingAPIResponse represents the SendGrid API response structure
type linkBrandingAPIResponse struct {
	ID        int                     `json:"id"`
	UserID    int                     `json:"user_id"`
	Domain    string                  `json:"domain"`
	Subdomain string                  `json:"subdomain"`
	Username  string                  `json:"username"`
	Default   bool                    `json:"default"`
	Valid     bool                    `json:"valid"`
	Legacy    bool                    `json:"legacy"`
	DNS       linkBrandingDNSResponse `json:"dns"`
}

type linkBrandingDNSResponse struct {
	OwnerCname linkBrandingDNSRecordResponse `json:"owner_cname"`
	BrandCname linkBrandingDNSRecordResponse `json:"brand_cname"`
}

type linkBrandingDNSRecordResponse struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"`
	Host  string `json:"host"`
	Data  string `json:"data"`
}

// toState converts an API response to LinkBrandingState
func (r *linkBrandingAPIResponse) toState() LinkBrandingState {
	state := LinkBrandingState{
		LinkBrandingArgs: LinkBrandingArgs{
			Domain: r.Domain,
		},
		LinkID:   r.ID,
		UserID:   r.UserID,
		Username: r.Username,
		Valid:    r.Valid,
		Legacy:   r.Legacy,
	}

	// Handle optional fields
	if r.Subdomain != "" {
		state.Subdomain = &r.Subdomain
	}
	if r.Default {
		state.Default = &r.Default
	}

	// Set DNS records
	if r.DNS.OwnerCname.Host != "" {
		state.OwnerCname = &LinkBrandingDNSRecord{
			Valid: r.DNS.OwnerCname.Valid,
			Type:  r.DNS.OwnerCname.Type,
			Host:  r.DNS.OwnerCname.Host,
			Data:  r.DNS.OwnerCname.Data,
		}
	}
	if r.DNS.BrandCname.Host != "" {
		state.BrandCname = &LinkBrandingDNSRecord{
			Valid: r.DNS.BrandCname.Valid,
			Type:  r.DNS.BrandCname.Type,
			Host:  r.DNS.BrandCname.Host,
			Data:  r.DNS.BrandCname.Data,
		}
	}

	return state
}

// Create creates a new SendGrid Link Branding.
func (l *LinkBranding) Create(ctx context.Context, req infer.CreateRequest[LinkBrandingArgs]) (infer.CreateResponse[LinkBrandingState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := LinkBrandingState{
			LinkBrandingArgs: input,
			LinkID:           0,
			UserID:           0,
			Valid:            false,
			Legacy:           false,
		}
		return infer.CreateResponse[LinkBrandingState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[LinkBrandingState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"domain": input.Domain,
	}

	// Add optional fields if provided
	if input.Subdomain != nil {
		reqBody["subdomain"] = *input.Subdomain
	}
	if input.Default != nil {
		reqBody["default"] = *input.Default
	}
	if input.Region != nil {
		reqBody["region"] = *input.Region
	}

	// Make the API call
	var result linkBrandingAPIResponse
	if err := client.Post(ctx, "/v3/whitelabel/links", reqBody, &result); err != nil {
		return infer.CreateResponse[LinkBrandingState]{}, fmt.Errorf("failed to create link branding: %w", err)
	}

	state := result.toState()

	return infer.CreateResponse[LinkBrandingState]{
		ID:     strconv.Itoa(result.ID),
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Link Branding.
func (l *LinkBranding) Read(ctx context.Context, req infer.ReadRequest[LinkBrandingArgs, LinkBrandingState]) (infer.ReadResponse[LinkBrandingArgs, LinkBrandingState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[LinkBrandingArgs, LinkBrandingState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the link branding details
	var result linkBrandingAPIResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/whitelabel/links/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[LinkBrandingArgs, LinkBrandingState]{}, nil
		}
		return infer.ReadResponse[LinkBrandingArgs, LinkBrandingState]{}, fmt.Errorf("failed to read link branding: %w", err)
	}

	state := result.toState()
	inputs := state.LinkBrandingArgs

	return infer.ReadResponse[LinkBrandingArgs, LinkBrandingState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Link Branding.
func (l *LinkBranding) Update(ctx context.Context, req infer.UpdateRequest[LinkBrandingArgs, LinkBrandingState]) (infer.UpdateResponse[LinkBrandingState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := LinkBrandingState{
			LinkBrandingArgs: input,
			LinkID:           oldState.LinkID,
			UserID:           oldState.UserID,
			Username:         oldState.Username,
			Valid:            oldState.Valid,
			Legacy:           oldState.Legacy,
			OwnerCname:       oldState.OwnerCname,
			BrandCname:       oldState.BrandCname,
		}
		return infer.UpdateResponse[LinkBrandingState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[LinkBrandingState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body - only default can be updated via PATCH
	reqBody := map[string]interface{}{}

	if input.Default != nil {
		reqBody["default"] = *input.Default
	}

	// Make the API call
	var result linkBrandingAPIResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/whitelabel/links/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[LinkBrandingState]{}, fmt.Errorf("failed to update link branding: %w", err)
	}

	state := result.toState()

	return infer.UpdateResponse[LinkBrandingState]{Output: state}, nil
}

// Delete removes a SendGrid Link Branding.
func (l *LinkBranding) Delete(ctx context.Context, req infer.DeleteRequest[LinkBrandingState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/whitelabel/links/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete link branding: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
