package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GrafanaService manages Grafana organizations and users for multi-tenant SaaS
type GrafanaService struct {
	baseURL  string
	apiToken string
	username string
	password string
	client   *http.Client
}

type GrafanaOrg struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CreateOrgRequest struct {
	Name string `json:"name"`
}

type CreateOrgResponse struct {
	OrgID   int    `json:"orgId"`
	Message string `json:"message"`
}

type AddUserToOrgRequest struct {
	LoginOrEmail string `json:"loginOrEmail"`
	Role         string `json:"role"` // Viewer, Editor, Admin
}

type GrafanaAPIError struct {
	Message string `json:"message"`
}

// NewGrafanaService creates a new Grafana service client
func NewGrafanaService(baseURL, apiToken, username, password string) *GrafanaService {
	return &GrafanaService{
		baseURL:  baseURL,
		apiToken: apiToken,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if Grafana service has valid credentials
func (s *GrafanaService) IsConfigured() bool {
	return s.baseURL != "" && (s.apiToken != "" || (s.username != "" && s.password != ""))
}

// request makes an authenticated request to Grafana API
func (s *GrafanaService) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := s.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Authentication
	if s.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiToken)
	} else if s.username != "" && s.password != "" {
		req.SetBasicAuth(s.username, s.password)
	} else {
		return nil, fmt.Errorf("no authentication credentials provided")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr GrafanaAPIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("grafana API error (%d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("grafana API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ListOrgs returns all organizations in Grafana
func (s *GrafanaService) ListOrgs(ctx context.Context) ([]GrafanaOrg, error) {
	respBody, err := s.request(ctx, "GET", "/api/orgs", nil)
	if err != nil {
		return nil, err
	}

	var orgs []GrafanaOrg
	if err := json.Unmarshal(respBody, &orgs); err != nil {
		return nil, fmt.Errorf("failed to parse orgs response: %w", err)
	}

	return orgs, nil
}

// GetOrgByName finds an organization by name
func (s *GrafanaService) GetOrgByName(ctx context.Context, name string) (*GrafanaOrg, error) {
	orgs, err := s.ListOrgs(ctx)
	if err != nil {
		return nil, err
	}

	for _, org := range orgs {
		if org.Name == name {
			return &org, nil
		}
	}

	return nil, nil // Not found
}

// CreateOrg creates a new organization
func (s *GrafanaService) CreateOrg(ctx context.Context, name string) (*CreateOrgResponse, error) {
	reqBody := CreateOrgRequest{Name: name}
	respBody, err := s.request(ctx, "POST", "/api/orgs", reqBody)
	if err != nil {
		return nil, err
	}

	var resp CreateOrgResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse create org response: %w", err)
	}

	return &resp, nil
}

// DeleteOrg deletes an organization
func (s *GrafanaService) DeleteOrg(ctx context.Context, orgID int) error {
	_, err := s.request(ctx, "DELETE", fmt.Sprintf("/api/orgs/%d", orgID), nil)
	return err
}

// CreateOrgForTenant creates a Grafana organization for a tenant
func (s *GrafanaService) CreateOrgForTenant(ctx context.Context, tenantID uint, tenantName string) (int, error) {
	orgName := fmt.Sprintf("Tenant %d - %s", tenantID, tenantName)

	// Check if org already exists
	existing, err := s.GetOrgByName(ctx, orgName)
	if err != nil {
		return 0, fmt.Errorf("failed to check existing org: %w", err)
	}
	if existing != nil {
		return existing.ID, nil // Already exists
	}

	// Create org
	resp, err := s.CreateOrg(ctx, orgName)
	if err != nil {
		return 0, fmt.Errorf("failed to create org: %w", err)
	}

	return resp.OrgID, nil
}

// AddUserToOrg adds a user to an organization with a specific role
func (s *GrafanaService) AddUserToOrg(ctx context.Context, orgID int, email string, role string) error {
	reqBody := AddUserToOrgRequest{
		LoginOrEmail: email,
		Role:         role,
	}

	_, err := s.request(ctx, "POST", fmt.Sprintf("/api/orgs/%d/users", orgID), reqBody)
	if err != nil {
		return fmt.Errorf("failed to add user to org: %w", err)
	}

	return nil
}

// RemoveUserFromOrg removes a user from an organization
func (s *GrafanaService) RemoveUserFromOrg(ctx context.Context, orgID int, userID int) error {
	_, err := s.request(ctx, "DELETE", fmt.Sprintf("/api/orgs/%d/users/%d", orgID, userID), nil)
	return err
}

// UpdateUserRole updates a user's role in an organization
func (s *GrafanaService) UpdateUserRole(ctx context.Context, orgID int, userID int, role string) error {
	reqBody := map[string]string{"role": role}
	_, err := s.request(ctx, "PATCH", fmt.Sprintf("/api/orgs/%d/users/%d", orgID, userID), reqBody)
	return err
}

// SyncTenantOrganization ensures a tenant has a corresponding Grafana organization
// and returns the organization ID
func (s *GrafanaService) SyncTenantOrganization(ctx context.Context, tenantID uint, tenantName string) (int, error) {
	return s.CreateOrgForTenant(ctx, tenantID, tenantName)
}
