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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultBaseURL is the default SendGrid API base URL
	DefaultBaseURL = "https://api.sendgrid.com"
)

// SendGridClient is an HTTP client for the SendGrid API
type SendGridClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewSendGridClient creates a new SendGrid API client
func NewSendGridClient(apiKey string, baseURL string) *SendGridClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &SendGridClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendGridError represents an error response from the SendGrid API
type SendGridError struct {
	StatusCode int
	Message    string
	Errors     []SendGridErrorDetail `json:"errors,omitempty"`
}

// SendGridErrorDetail represents a detailed error from SendGrid
type SendGridErrorDetail struct {
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Help    string `json:"help,omitempty"`
}

func (e *SendGridError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("SendGrid API error (status %d): %s", e.StatusCode, e.Errors[0].Message)
	}
	return fmt.Sprintf("SendGrid API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found
func (e *SendGridError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// doRequest performs an HTTP request to the SendGrid API
func (c *SendGridClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		sgErr := &SendGridError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
		}
		// Try to parse the error response
		if len(respBody) > 0 {
			var errResp struct {
				Errors []SendGridErrorDetail `json:"errors"`
			}
			if json.Unmarshal(respBody, &errResp) == nil && len(errResp.Errors) > 0 {
				sgErr.Errors = errResp.Errors
			}
		}
		return sgErr
	}

	// Parse successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request
func (c *SendGridClient) Get(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request
func (c *SendGridClient) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request
func (c *SendGridClient) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPut, path, body, result)
}

// Patch performs a PATCH request
func (c *SendGridClient) Patch(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPatch, path, body, result)
}

// Delete performs a DELETE request
func (c *SendGridClient) Delete(ctx context.Context, path string) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}
