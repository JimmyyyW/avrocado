package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/JimmyyyW/avrocado/internal/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	apiSecret  string
}

type SchemaResponse struct {
	Subject    string `json:"subject"`
	Version    int    `json:"version"`
	ID         int    `json:"id"`
	SchemaType string `json:"schemaType"`
	Schema     string `json:"schema"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(cfg.RegistryURL, "/"),
		httpClient: &http.Client{},
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
	}
}

func (c *Client) doRequest(method, path string) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")

	if c.apiKey != "" && c.apiSecret != "" {
		req.SetBasicAuth(c.apiKey, c.apiSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) ListSubjects() ([]string, error) {
	body, err := c.doRequest(http.MethodGet, "/subjects")
	if err != nil {
		return nil, err
	}

	var subjects []string
	if err := json.Unmarshal(body, &subjects); err != nil {
		return nil, fmt.Errorf("parsing subjects: %w", err)
	}

	return subjects, nil
}

func (c *Client) GetLatestSchema(subject string) (*SchemaResponse, error) {
	path := fmt.Sprintf("/subjects/%s/versions/latest", subject)
	body, err := c.doRequest(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var schema SchemaResponse
	if err := json.Unmarshal(body, &schema); err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	return &schema, nil
}

func PrettyPrintSchema(schema string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		return schema
	}

	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return schema
	}

	return string(pretty)
}
