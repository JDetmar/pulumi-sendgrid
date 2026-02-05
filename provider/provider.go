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

// Package provider implements the SendGrid Pulumi provider.
package provider

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Version is initialized by the Go linker to contain the semver of this build.
var Version string

// Name controls how this provider is referenced in package names and elsewhere.
const Name string = "sendgrid"

// Provider creates a new instance of the SendGrid provider.
func Provider() p.Provider {
	prov, err := infer.NewProviderBuilder().
		WithDisplayName("SendGrid").
		WithDescription("A Pulumi provider for managing SendGrid resources.").
		WithHomepage("https://www.pulumi.com").
		WithNamespace("pulumi").
		WithResources(
			infer.Resource(&ApiKey{}),
		).
		WithConfig(infer.Config(&Config{})).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		}).Build()
	if err != nil {
		panic(fmt.Errorf("unable to build provider: %w", err))
	}
	return prov
}

// Config defines provider-level configuration for SendGrid.
type Config struct {
	// ApiKey is the SendGrid API key used for authentication.
	// Can also be set via the SENDGRID_API_KEY environment variable.
	ApiKey *string `pulumi:"apiKey,optional" provider:"secret"`

	// BaseURL is the SendGrid API base URL. Defaults to https://api.sendgrid.com.
	// Can be overridden for testing or for EU regional endpoints.
	BaseURL *string `pulumi:"baseUrl,optional"`

	// client is the initialized SendGrid client (not exposed to Pulumi)
	client *SendGridClient
}

// Annotate provides descriptions for the Config fields.
func (c *Config) Annotate(annotator infer.Annotator) {
	annotator.Describe(&c.ApiKey, "The SendGrid API key for authentication. "+
		"Can also be set via the SENDGRID_API_KEY environment variable.")
	annotator.Describe(&c.BaseURL, "The SendGrid API base URL. "+
		"Defaults to https://api.sendgrid.com. Use https://api.eu.sendgrid.com for EU regional subusers.")
	annotator.SetDefault(&c.BaseURL, DefaultBaseURL)
}

// Configure initializes the SendGrid client based on the provided configuration.
func (c *Config) Configure(ctx context.Context) error {
	// Get API key from config or environment
	apiKey := ""
	if c.ApiKey != nil && *c.ApiKey != "" {
		apiKey = *c.ApiKey
	} else {
		apiKey = os.Getenv("SENDGRID_API_KEY")
	}

	if apiKey == "" {
		return fmt.Errorf("SendGrid API key is required. Set it via the 'apiKey' provider config or SENDGRID_API_KEY environment variable")
	}

	// Get base URL from config or use default
	baseURL := DefaultBaseURL
	if c.BaseURL != nil && *c.BaseURL != "" {
		baseURL = *c.BaseURL
	}

	// Initialize the client
	c.client = NewSendGridClient(apiKey, baseURL)

	return nil
}
