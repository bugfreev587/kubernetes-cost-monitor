package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const clerkAPIBaseURL = "https://api.clerk.com/v1"

// ClerkService handles interactions with Clerk's Backend API
type ClerkService struct {
	secretKey   string
	frontendURL string
	httpClient  *http.Client
}

// NewClerkService creates a new Clerk service
func NewClerkService(secretKey, frontendURL string) *ClerkService {
	return &ClerkService{
		secretKey:   secretKey,
		frontendURL: frontendURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if Clerk is properly configured
func (s *ClerkService) IsConfigured() bool {
	return s.secretKey != ""
}

// GetFrontendURL returns the frontend URL configured for this service
func (s *ClerkService) GetFrontendURL() string {
	return s.frontendURL
}

// InvitationRequest represents a request to create an invitation
type InvitationRequest struct {
	EmailAddress   string                 `json:"email_address"`
	PublicMetadata map[string]interface{} `json:"public_metadata,omitempty"`
	RedirectURL    string                 `json:"redirect_url,omitempty"`
	Notify         bool                   `json:"notify"` // If true, Clerk sends the invitation email
	IgnoreExisting bool                   `json:"ignore_existing,omitempty"`
}

// InvitationResponse represents a response from Clerk's invitation API
type InvitationResponse struct {
	ID             string                 `json:"id"`
	EmailAddress   string                 `json:"email_address"`
	PublicMetadata map[string]interface{} `json:"public_metadata"`
	Status         string                 `json:"status"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
}

// ClerkError represents an error response from Clerk
type ClerkError struct {
	Errors []struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		LongMsg string                 `json:"long_message"`
		Meta    map[string]interface{} `json:"meta,omitempty"`
	} `json:"errors"`
}

func (e *ClerkError) Error() string {
	if len(e.Errors) > 0 {
		return e.Errors[0].Message
	}
	return "unknown clerk error"
}

// CreateInvitation creates an invitation in Clerk and sends an email
func (s *ClerkService) CreateInvitation(ctx context.Context, email string, tenantID uint, role string, inviterName string) (*InvitationResponse, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("clerk is not configured (missing secret key)")
	}

	// Build redirect URL - user will be redirected here after accepting
	redirectURL := s.frontendURL
	if redirectURL == "" {
		redirectURL = "https://my-k8s-cost-monitor.dedyn.io"
	}
	redirectURL = redirectURL + "/sign-up"

	// Include metadata so we can link the user to the tenant when they sign up
	metadata := map[string]interface{}{
		"tenant_id":    tenantID,
		"role":         role,
		"invited":      true,
		"inviter_name": inviterName,
	}

	req := InvitationRequest{
		EmailAddress:   email,
		PublicMetadata: metadata,
		RedirectURL:    redirectURL,
		Notify:         true, // Send invitation email
		IgnoreExisting: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invitation request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", clerkAPIBaseURL+"/invitations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send invitation request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var clerkErr ClerkError
		if err := json.Unmarshal(respBody, &clerkErr); err == nil && len(clerkErr.Errors) > 0 {
			log.Printf("Clerk API error: %s - %s", clerkErr.Errors[0].Code, clerkErr.Errors[0].Message)
			return nil, &clerkErr
		}
		return nil, fmt.Errorf("clerk API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var invResp InvitationResponse
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return nil, fmt.Errorf("failed to parse invitation response: %w", err)
	}

	log.Printf("Clerk invitation created: id=%s, email=%s, status=%s", invResp.ID, invResp.EmailAddress, invResp.Status)

	return &invResp, nil
}

// RevokeInvitation revokes a pending invitation
func (s *ClerkService) RevokeInvitation(ctx context.Context, invitationID string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("clerk is not configured (missing secret key)")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", clerkAPIBaseURL+"/invitations/"+invitationID+"/revoke", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.secretKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send revoke request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to revoke invitation: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	log.Printf("Clerk invitation revoked: id=%s", invitationID)
	return nil
}

// RevokeUserSessions revokes all active sessions for a user, forcing them to log out
// This should be called when a user is removed from a tenant
func (s *ClerkService) RevokeUserSessions(ctx context.Context, clerkUserID string) error {
	if !s.IsConfigured() {
		log.Printf("Clerk not configured, skipping session revocation for user %s", clerkUserID)
		return fmt.Errorf("clerk is not configured (missing secret key)")
	}

	log.Printf("Revoking sessions for user %s", clerkUserID)

	// Get all sessions for the user using query parameter
	httpReq, err := http.NewRequestWithContext(ctx, "GET", clerkAPIBaseURL+"/sessions?user_id="+clerkUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.secretKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to get sessions for user %s: status=%d, body=%s", clerkUserID, resp.StatusCode, string(respBody))
		return fmt.Errorf("failed to get user sessions: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// Parse sessions response - Clerk returns an array directly
	var sessions []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &sessions); err != nil {
		log.Printf("Failed to parse sessions response: %v, body: %s", err, string(respBody))
		return fmt.Errorf("failed to parse sessions response: %w", err)
	}

	log.Printf("Found %d sessions for user %s", len(sessions), clerkUserID)

	// Revoke each active session
	revokedCount := 0
	for _, session := range sessions {
		// Only revoke active sessions
		if session.Status != "active" {
			log.Printf("Skipping non-active session %s (status: %s)", session.ID, session.Status)
			continue
		}

		revokeReq, err := http.NewRequestWithContext(ctx, "POST", clerkAPIBaseURL+"/sessions/"+session.ID+"/revoke", nil)
		if err != nil {
			log.Printf("Warning: Failed to create revoke request for session %s: %v", session.ID, err)
			continue
		}

		revokeReq.Header.Set("Authorization", "Bearer "+s.secretKey)

		revokeResp, err := s.httpClient.Do(revokeReq)
		if err != nil {
			log.Printf("Warning: Failed to revoke session %s: %v", session.ID, err)
			continue
		}

		revokeBody, _ := io.ReadAll(revokeResp.Body)
		revokeResp.Body.Close()

		if revokeResp.StatusCode != http.StatusOK {
			log.Printf("Warning: Failed to revoke session %s: status=%d, body=%s", session.ID, revokeResp.StatusCode, string(revokeBody))
			continue
		}

		log.Printf("Revoked session %s for user %s", session.ID, clerkUserID)
		revokedCount++
	}

	log.Printf("Successfully revoked %d sessions for user %s", revokedCount, clerkUserID)
	return nil
}

// UpdateUserMetadata updates a user's public metadata in Clerk
// This is required for Grafana OAuth integration to work properly
func (s *ClerkService) UpdateUserMetadata(ctx context.Context, clerkUserID string, tenantID uint, role string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("clerk is not configured (missing secret key)")
	}

	// Build the metadata to set
	metadata := map[string]interface{}{
		"tenant_id": tenantID,
		"role":      role,
		"roles":     []string{role}, // Grafana expects an array for role mapping
	}

	reqBody := map[string]interface{}{
		"public_metadata": metadata,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", clerkAPIBaseURL+"/users/"+clerkUserID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var clerkErr ClerkError
		if err := json.Unmarshal(respBody, &clerkErr); err == nil && len(clerkErr.Errors) > 0 {
			return fmt.Errorf("clerk API error: %s - %s", clerkErr.Errors[0].Code, clerkErr.Errors[0].Message)
		}
		return fmt.Errorf("failed to update user metadata: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	log.Printf("Updated Clerk user metadata: user_id=%s, tenant_id=%d, role=%s", clerkUserID, tenantID, role)
	return nil
}

// ClearUserMetadata removes tenant and role metadata from a user in Clerk
// This should be called when a user is removed from a tenant to prevent
// unauthorized access to Grafana via OAuth
func (s *ClerkService) ClearUserMetadata(ctx context.Context, clerkUserID string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("clerk is not configured (missing secret key)")
	}

	// Set metadata fields to null to clear them
	metadata := map[string]interface{}{
		"tenant_id": nil,
		"role":      nil,
		"roles":     nil,
	}

	reqBody := map[string]interface{}{
		"public_metadata": metadata,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal clear metadata request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", clerkAPIBaseURL+"/users/"+clerkUserID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send clear metadata request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var clerkErr ClerkError
		if err := json.Unmarshal(respBody, &clerkErr); err == nil && len(clerkErr.Errors) > 0 {
			return fmt.Errorf("clerk API error: %s - %s", clerkErr.Errors[0].Code, clerkErr.Errors[0].Message)
		}
		return fmt.Errorf("failed to clear user metadata: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	log.Printf("Cleared Clerk user metadata: user_id=%s", clerkUserID)
	return nil
}

// ListInvitations lists all pending invitations
func (s *ClerkService) ListInvitations(ctx context.Context, status string) ([]InvitationResponse, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("clerk is not configured (missing secret key)")
	}

	url := clerkAPIBaseURL + "/invitations"
	if status != "" {
		url += "?status=" + status
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.secretKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list invitations: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var invitations []InvitationResponse
	if err := json.NewDecoder(resp.Body).Decode(&invitations); err != nil {
		return nil, fmt.Errorf("failed to parse invitations response: %w", err)
	}

	return invitations, nil
}
