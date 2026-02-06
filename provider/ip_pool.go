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

// IpPool is the controller for the SendGrid IP Pool resource.
//
// This resource manages SendGrid IP Pools, which allow you to group
// your dedicated IP addresses. For example, you could have a pool
// for transactional emails and another for marketing emails.
type IpPool struct{} //nolint:revive // name matches Pulumi resource token

// IpPoolArgs are the inputs to the IpPool resource.
type IpPoolArgs struct { //nolint:revive // name matches Pulumi resource token
	// Name is the name of the IP pool (required, max 64 chars)
	Name string `pulumi:"name"`
}

// IpPoolState is the state of the IpPool resource.
type IpPoolState struct { //nolint:revive // name matches Pulumi resource token
	// Embed the input args in the output state
	IpPoolArgs

	// PoolName is the name of the IP pool (returned by API)
	PoolName string `pulumi:"poolName"`

	// Ips is the list of IP addresses assigned to this pool
	Ips []string `pulumi:"ips"`
}

// Annotate provides descriptions for the IpPool resource.
func (p *IpPool) Annotate(annotator infer.Annotator) {
	annotator.Describe(&p, "Manages a SendGrid IP Pool.\n\n"+
		"IP Pools allow you to group your dedicated SendGrid IP addresses together. "+
		"For example, you might have separate pools for transactional and marketing emails, "+
		"so that each pool maintains its own reputation.\n\n"+
		"Note: Each account can create up to 100 IP pools. IP pools can only be used with "+
		"IP addresses that have reverse DNS configured.")
}

// ipPoolAPIResponse represents the SendGrid API response structure for IP pools
type ipPoolAPIResponse struct {
	PoolName string   `json:"pool_name"`
	Name     string   `json:"name,omitempty"`
	Ips      []string `json:"ips"`
}

// toState converts an API response to IpPoolState
func (r *ipPoolAPIResponse) toState() IpPoolState {
	name := r.PoolName
	if name == "" {
		name = r.Name
	}
	return IpPoolState{
		IpPoolArgs: IpPoolArgs{
			Name: name,
		},
		PoolName: name,
		Ips:      r.Ips,
	}
}

// Create creates a new SendGrid IP Pool.
func (p *IpPool) Create(ctx context.Context, req infer.CreateRequest[IpPoolArgs]) (infer.CreateResponse[IpPoolState], error) {
	input := req.Inputs
	preview := req.DryRun

	// During preview, return placeholder state
	if preview {
		state := IpPoolState{
			IpPoolArgs: input,
			PoolName:   input.Name,
			Ips:        []string{},
		}
		return infer.CreateResponse[IpPoolState]{
			ID:     "[preview]",
			Output: state,
		}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.CreateResponse[IpPoolState]{}, fmt.Errorf("SendGrid client not configured - ensure apiKey is set in provider configuration")
	}

	// Build the request body
	reqBody := map[string]interface{}{
		"name": input.Name,
	}

	// Make the API call
	var result ipPoolAPIResponse
	if err := client.Post(ctx, "/v3/ips/pools", reqBody, &result); err != nil {
		return infer.CreateResponse[IpPoolState]{}, fmt.Errorf("failed to create IP pool: %w", err)
	}

	state := result.toState()

	// Use pool_name as the ID (URL encoded for safety)
	return infer.CreateResponse[IpPoolState]{
		ID:     state.PoolName,
		Output: state,
	}, nil
}

// Read retrieves the current state of a SendGrid IP Pool.
func (p *IpPool) Read(ctx context.Context, req infer.ReadRequest[IpPoolArgs, IpPoolState]) (infer.ReadResponse[IpPoolArgs, IpPoolState], error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.ReadResponse[IpPoolArgs, IpPoolState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL encode the pool name for the path
	encodedName := url.PathEscape(id)

	// Make the API call to get the IP pool details
	var result ipPoolAPIResponse
	if err := client.Get(ctx, fmt.Sprintf("/v3/ips/pools/%s", encodedName), &result); err != nil {
		// Check if the resource was deleted out-of-band
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			// Return empty response to indicate resource no longer exists
			return infer.ReadResponse[IpPoolArgs, IpPoolState]{}, nil
		}
		return infer.ReadResponse[IpPoolArgs, IpPoolState]{}, fmt.Errorf("failed to read IP pool: %w", err)
	}

	state := result.toState()
	inputs := state.IpPoolArgs

	return infer.ReadResponse[IpPoolArgs, IpPoolState]{
		ID:     id,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update updates an existing SendGrid IP Pool.
func (p *IpPool) Update(ctx context.Context, req infer.UpdateRequest[IpPoolArgs, IpPoolState]) (infer.UpdateResponse[IpPoolState], error) {
	id := req.ID
	input := req.Inputs
	oldState := req.State
	preview := req.DryRun

	// During preview, return expected state
	if preview {
		state := IpPoolState{
			IpPoolArgs: input,
			PoolName:   input.Name,
			Ips:        oldState.Ips,
		}
		return infer.UpdateResponse[IpPoolState]{Output: state}, nil
	}

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.UpdateResponse[IpPoolState]{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL encode the pool name for the path
	encodedName := url.PathEscape(id)

	// Build the request body - update pool name
	reqBody := map[string]interface{}{
		"name": input.Name,
	}

	// Make the API call (PUT to update pool name)
	var result ipPoolAPIResponse
	if err := client.Put(ctx, fmt.Sprintf("/v3/ips/pools/%s", encodedName), reqBody, &result); err != nil {
		return infer.UpdateResponse[IpPoolState]{}, fmt.Errorf("failed to update IP pool: %w", err)
	}

	state := result.toState()

	return infer.UpdateResponse[IpPoolState]{Output: state}, nil
}

// Delete removes a SendGrid IP Pool.
func (p *IpPool) Delete(ctx context.Context, req infer.DeleteRequest[IpPoolState]) (infer.DeleteResponse, error) {
	id := req.ID

	// Get the SendGrid client from context
	client := infer.GetConfig[Config](ctx).client
	if client == nil {
		return infer.DeleteResponse{}, fmt.Errorf("SendGrid client not configured")
	}

	// URL encode the pool name for the path
	encodedName := url.PathEscape(id)

	// Make the API call
	if err := client.Delete(ctx, fmt.Sprintf("/v3/ips/pools/%s", encodedName)); err != nil {
		// If already deleted, that's fine
		if sgErr, ok := err.(*SendGridError); ok && sgErr.IsNotFound() {
			return infer.DeleteResponse{}, nil
		}
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete IP pool: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
