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
		WithHomepage("https://github.com/JDetmar/pulumi-sendgrid").
		WithRepository("https://github.com/JDetmar/pulumi-sendgrid").
		WithPluginDownloadURL("github://api.github.com/JDetmar/pulumi-sendgrid").
		WithNamespace("pulumi").
		WithLanguageMap(map[string]any{
			"csharp": map[string]any{
				"rootNamespace":        "Community.Pulumi",
				"respectSchemaVersion": true,
			},
			"java": map[string]any{
				"basePackage": "io.github.jdetmar.pulumi",
				"buildFiles":  "gradle",
			},
			"nodejs": map[string]any{
				"packageName":          "@jdetmar/pulumi-sendgrid",
				"packageDescription":   "A Pulumi provider for managing SendGrid resources.",
				"respectSchemaVersion": true,
			},
			"python": map[string]any{
				"packageName":        "pulumi_sendgrid",
				"packageDescription": "A Pulumi provider for managing SendGrid resources.",
			},
		}).
		WithResources(
			infer.Resource(&ApiKey{}),
			infer.Resource(&Template{}),
			infer.Resource(&TemplateVersion{}),
			infer.Resource(&VerifiedSender{}),
			infer.Resource(&DomainAuthentication{}),
			infer.Resource(&LinkBranding{}),
			infer.Resource(&IpPool{}),
			infer.Resource(&UnsubscribeGroup{}),
			infer.Resource(&GlobalSuppression{}),
			infer.Resource(&EventWebhook{}),
			infer.Resource(&Subuser{}),
			infer.Resource(&Teammate{}),
			infer.Resource(&Alert{}),
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
	// APIKey is the SendGrid API key used for authentication.
	// Can also be set via the SENDGRID_API_KEY environment variable.
	APIKey *string `pulumi:"apiKey,optional" provider:"secret"`

	// BaseURL is the SendGrid API base URL. Defaults to https://api.sendgrid.com.
	// Can be overridden for testing or for EU regional endpoints.
	BaseURL *string `pulumi:"baseUrl,optional"`

	// client is the initialized SendGrid client (not exposed to Pulumi)
	client *SendGridClient
}

// Annotate provides descriptions for the Config fields.
func (c *Config) Annotate(annotator infer.Annotator) {
	annotator.Describe(&c.APIKey, "The SendGrid API key for authentication. "+
		"Can also be set via the SENDGRID_API_KEY environment variable.")
	annotator.Describe(&c.BaseURL, "The SendGrid API base URL. "+
		"Defaults to https://api.sendgrid.com. Use https://api.eu.sendgrid.com for EU regional subusers.")
	annotator.SetDefault(&c.BaseURL, DefaultBaseURL)
}

// Configure initializes the SendGrid client based on the provided configuration.
func (c *Config) Configure(_ context.Context) error {
	// Get API key from config or environment
	apiKey := ""
	if c.APIKey != nil && *c.APIKey != "" {
		apiKey = *c.APIKey
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
