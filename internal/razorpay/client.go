// Package razorpay is a small, dependency-free client for the Razorpay REST API.
// It keeps the raw JSON of every entity so we can store exactly what Razorpay sent.
package razorpay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.razorpay.com/v1"

type Client struct {
	keyID     string
	keySecret string
	baseURL   string
	http      *http.Client
}

func NewClient(keyID, keySecret string) *Client {
	return &Client{
		keyID:     keyID,
		keySecret: keySecret,
		baseURL:   defaultBaseURL,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) KeyID() string     { return c.keyID }
func (c *Client) KeySecret() string { return c.keySecret }

// APIError is Razorpay's error envelope: {"error": {...}}.
type APIError struct {
	StatusCode  int    `json:"status_code"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
	Step        string `json:"step,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Field       string `json:"field,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("razorpay %d %s: %s", e.StatusCode, e.Code, e.Description)
}

// RawSetter is implemented by every entity type (via the embedded withRaw).
type RawSetter interface {
	SetRaw([]byte)
}

type withRaw struct {
	Raw json.RawMessage `json:"-"`
}

func (w *withRaw) SetRaw(b []byte) { w.Raw = append(json.RawMessage(nil), b...) }

// DecodeEntity unmarshals Razorpay JSON into out and keeps a copy of the raw bytes.
func DecodeEntity(data []byte, out RawSetter) error {
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode razorpay entity: %w", err)
	}
	out.SetRaw(data)
	return nil
}

func (c *Client) send(ctx context.Context, method, path string, body any, keyOnlyAuth bool) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if keyOnlyAuth {
		req.SetBasicAuth(c.keyID, "")
	} else {
		req.SetBasicAuth(c.keyID, c.keySecret)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("razorpay request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var envelope struct {
			Error APIError `json:"error"`
		}
		_ = json.Unmarshal(data, &envelope)
		apiErr := envelope.Error
		apiErr.StatusCode = resp.StatusCode
		if apiErr.Description == "" {
			apiErr.Description = string(data)
		}
		return nil, &apiErr
	}
	return data, nil
}

// call performs a request and decodes a single entity of type T.
func call[T any, PT interface {
	*T
	RawSetter
}](c *Client, ctx context.Context, method, path string, body any) (*T, error) {
	data, err := c.send(ctx, method, path, body, false)
	if err != nil {
		return nil, err
	}
	var v T
	if err := DecodeEntity(data, PT(&v)); err != nil {
		return nil, err
	}
	return &v, nil
}

// list performs a request that returns a collection ({"count": n, "items": [...]}).
// It returns the decoded items plus the raw response body.
func list[T any, PT interface {
	*T
	RawSetter
}](c *Client, ctx context.Context, path string) ([]*T, json.RawMessage, error) {
	data, err := c.send(ctx, http.MethodGet, path, nil, false)
	if err != nil {
		return nil, nil, err
	}
	var coll struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, nil, fmt.Errorf("decode collection: %w", err)
	}
	out := make([]*T, 0, len(coll.Items))
	for _, item := range coll.Items {
		var v T
		if err := DecodeEntity(item, PT(&v)); err != nil {
			return nil, nil, err
		}
		out = append(out, &v)
	}
	return out, data, nil
}
