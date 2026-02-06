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

// DomainAuthentication is the controller for the SendGrid Domain Authentication resource.
//
// This resource manages SendGrid Domain Authentication (formerly "Domain Whitelabel"),
// which allows you to remove the "via" or "sent on behalf of" message in emails.
type DomainAuthentication struct{}

// DomainAuthenticationArgs are the inputs to the DomainAuthentication resource.
type DomainAuthenticationArgs struct {
	// Domain is the domain being authenticated (required)
	Domain string `pulumi:"domain"`

	// Subdomain is the subdomain to use for the authenticated domain (optional)
	// This is the custom return-path for the domain.
	Subdomain *string `pulumi:"subdomain,optional"`

	// Ips is a list of IP addresses to associate with this domain for custom SPF (optional)
	Ips []string `pulumi:"ips,optional"`

	// CustomSpf enables custom SPF record instead of SendGrid-managed (optional)
	CustomSpf *bool `pulumi:"customSpf,optional"`

	// Default marks this domain as the fallback/default domain (optional)
	Default *bool `pulumi:"default,optional"`

	// AutomaticSecurity allows SendGrid to automatically manage SPF and DKIM records (optional)
	// When enabled, SendGrid provides CNAME records. When disabled, you get TXT and MX records.
	AutomaticSecurity *bool `pulumi:"automaticSecurity,optional"`

	// CustomDkimSelector is a custom DKIM selector (max 3 characters) (optional)
	CustomDkimSelector *string `pulumi:"customDkimSelector,optional"`

	// Region is the region for the domain: "global" or "eu" (optional, default: global)
	Region *string `pulumi:"region,optional"`
}

// DNSRecord represents a DNS record required for domain authentication
type DNSRecord struct {
	// Valid indicates if the record has been validated
	Valid bool `pulumi:"valid"`
	// Type is the DNS record type (CNAME, TXT, MX)
	Type string `pulumi:"type"`
	// Host is the hostname for the record
	Host string `pulumi:"host"`
	// Data is the value/data for the record
	Data string `pulumi:"data"`
}

// DomainAuthenticationState is the state of the DomainAuthentication resource.
type DomainAuthenticationState struct {
	// Embed the input args in the output state
	DomainAuthenticationArgs

	// DomainID is the unique identifier for this authenticated domain
	DomainID int `pulumi:"domainId"`

	// UserID is the ID of the user that this domain is associated with
	UserID int `pulumi:"userId"`

	// Username is the username associated with this domain
	Username string `pulumi:"username"`

	// Valid indicates whether the domain has been validated
	Valid bool `pulumi:"valid"`

	// Legacy indicates if this is a legacy whitelabel
	Legacy bool `pulumi:"legacy"`

	// MailCname is the CNAME record for mail
	MailCname *DNSRecord `pulumi:"mailCname,optional"`

	// Dkim1 is the first DKIM record
	Dkim1 *DNSRecord `pulumi:"dkim1,optional"`

	// Dkim2 is the second DKIM record
	Dkim2 *DNSRecord `pulumi:"dkim2,optional"`
}

// Annotate provides descriptions for the DomainAuthentication resource.
func (d *DomainAuthentication) Annotate(annotator infer.Annotator) {
	annotator.Describe(&d, "Manages a SendGrid Domain Authentication.\n\n"+
		"Domain Authentication (formerly Domain Whitelabel) allows you to authenticate "+
		"your domain so that emails appear to come directly from your domain, "+
		"removing the 'via sendgrid.net' message that recipients may see.\n\n"+
		"After creating this resource, you must add the DNS records to your domain's DNS settings "+
		"and then validate the domain using the SendGrid console or API.")
}

// domainAuthAPIResponse represents the SendGrid API response structure
type domainAuthAPIResponse struct {
	ID                int                   `json:"id"`
	UserID            int                   `json:"user_id"`
	Domain            string                `json:"domain"`
	Subdomain         string                `json:"subdomain"`
	Username          string                `json:"username"`
	Ips               []string              `json:"ips"`
	CustomSpf         bool                  `json:"custom_spf"`
	Default           bool                  `json:"default"`
	AutomaticSecurity bool                  `json:"automatic_security"`
	Valid             bool                  `json:"valid"`
	Legacy            bool                  `json:"legacy"`
	DNS               domainAuthDNSResponse `json:"dns"`
}

type domainAuthDNSResponse struct {
	MailCname dnsRecordResponse `json:"mail_cname"`
	Dkim1     dnsRecordResponse `json:"dkim1"`
	Dkim2     dnsRecordResponse `json:"dkim2"`
}

type dnsRecordResponse struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"`
	Host  string `json:"host"`
	Data  string `json:"data"`
}

// toState converts an API response to DomainAuthenticationState
func (r *domainAuthAPIResponse) toState() DomainAuthenticationState {
	state := DomainAuthenticationState{
		DomainAuthenticationArgs: DomainAuthenticationArgs{
			Domain: r.Domain,
			Ips:    r.Ips,
		},
		DomainID: r.ID,
		UserID:   r.UserID,
		Username: r.Username,
		Valid:    r.Valid,
		Legacy:   r.Legacy,
	}

	// Handle optional fields
	if r.Subdomain != "" {
		state.Subdomain = &r.Subdomain
	}
	if r.CustomSpf {
		state.CustomSpf = &r.CustomSpf
	}
	if r.Default {
		state.Default = &r.Default
	}
	if r.AutomaticSecurity {
		state.AutomaticSecurity = &r.AutomaticSecurity
	}

	// Set DNS records
	if r.DNS.MailCname.Host != "" {
		state.MailCname = &DNSRecord{
			Valid: r.DNS.MailCname.Valid,
			Type:  r.DNS.MailCname.Type,
			Host:  r.DNS.MailCname.Host,
			Data:  r.DNS.MailCname.Data,
		}
	}
	if r.DNS.Dkim1.Host != "" {
		state.Dkim1 = &DNSRecord{
			Valid: r.DNS.Dkim1.Valid,
			Type:  r.DNS.Dkim1.Type,
			Host:  r.DNS.Dkim1.Host,
			Data:  r.DNS.Dkim1.Data,
		}
	}
	if r.DNS.Dkim2.Host != "" {
		state.Dkim2 = &DNSRecord{
			Valid: r.DNS.Dkim2.Valid,
			Type:  r.DNS.Dkim2.Type,
			Host:  r.DNS.Dkim2.Host,
			Data:  r.DNS.Dkim2.Data,
		}
	}

	return state
}

// Create creates a new SendGrid Domain Authentication.
func (d *DomainAuthentication) Create(ctx context.Context, req infer.CreateRequest[DomainAuthenticationArgs]) (infer.CreateResponse[DomainAuthenticationState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := DomainAuthenticationState{
			DomainAuthenticationArgs: input,
			DomainID:                 0,
			UserID:                   0,
			Valid:                    false,
			Legacy:                   false,
		}
		return infer.CreateResponse[DomainAuthenticationState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[DomainAuthenticationState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"domain": input.Domain,
	}

	// Add optional fields if provided
	if input.Subdomain != nil {
		reqBody["subdomain"] = *input.Subdomain
	}
	if len(input.Ips) > 0 {
		reqBody["ips"] = input.Ips
	}
	if input.CustomSpf != nil {
		reqBody["custom_spf"] = *input.CustomSpf
	}
	if input.Default != nil {
		reqBody["default"] = *input.Default
	}
	if input.AutomaticSecurity != nil {
		reqBody["automatic_security"] = *input.AutomaticSecurity
	}
	if input.CustomDkimSelector != nil {
		reqBody["custom_dkim_selector"] = *input.CustomDkimSelector
	}
	if input.Region != nil {
		reqBody["region"] = *input.Region
	}

	// Make the API call
	var result domainAuthAPIResponse
	if err := client.Post(ctx, "/v3/whitelabel/domains", reqBody, &result); err != nil {
		return infer.CreateResponse[DomainAuthenticationState]{}, fmt.Errorf("failed to create domain authentication: %w", err)
	}

	state := result.toState()

	return infer.CreateResponse[DomainAuthenticationState]{
		ID:     strconv.Itoa(result.ID),
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid Domain Authentication.
func (d *DomainAuthentication) Read(ctx context.Context, req infer.ReadRequest[DomainAuthenticationArgs, DomainAuthenticationState]) (infer.ReadResponse[DomainAuthenticationArgs, DomainAuthenticationState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[DomainAuthenticationArgs, DomainAuthenticationState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call to get the domain authentication details
	var result domainAuthAPIResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/whitelabel/domains/%s", id), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[DomainAuthenticationArgs, DomainAuthenticationState]{}, nil
		}
		return infer.ReadResponse[DomainAuthenticationArgs, DomainAuthenticationState]{}, fmt.Errorf("failed to read domain authentication: %w", err)
	}

	state := result.toState()
	inputs := state.DomainAuthenticationArgs

	return infer.ReadResponse[DomainAuthenticationArgs, DomainAuthenticationState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid Domain Authentication.
func (d *DomainAuthentication) Update(ctx context.Context, req infer.UpdateRequest[DomainAuthenticationArgs, DomainAuthenticationState]) (infer.UpdateResponse[DomainAuthenticationState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := DomainAuthenticationState{
			DomainAuthenticationArgs: input,
			DomainID:                 oldState.DomainID,
			UserID:                   oldState.UserID,
			Username:                 oldState.Username,
			Valid:                    oldState.Valid,
			Legacy:                   oldState.Legacy,
			MailCname:                oldState.MailCname,
			Dkim1:                    oldState.Dkim1,
			Dkim2:                    oldState.Dkim2,
		}
		return infer.UpdateResponse[DomainAuthenticationState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[DomainAuthenticationState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// Build the request body - only default and custom_spf can be updated
	reqBody := map[string]interface{}{}

	if input.Default != nil {
		reqBody["default"] = *input.Default
	}
	if input.CustomSpf != nil {
		reqBody["custom_spf"] = *input.CustomSpf
	}

	// Make the API call
	var result domainAuthAPIResponse
	if err := client.Patch(ctx, fmt.Sprintf("/v3/whitelabel/domains/%s", id), reqBody, &result); err != nil {
		return infer.UpdateResponse[DomainAuthenticationState]{}, fmt.Errorf("failed to update domain authentication: %w", err)
	}

	state := result.toState()

	return infer.UpdateResponse[DomainAuthenticationState]{Output: state}, nil
}

// Delete removes a SendGrid Domain Authentication.
func (d *DomainAuthentication) Delete(ctx context.Context, req infer.DeleteRequest[DomainAuthenticationState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/whitelabel/domains/%s", id)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete domain authentication: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
