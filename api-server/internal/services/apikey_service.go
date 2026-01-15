package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
)

type APIKeyService struct {
	db       *gorm.DB
	pepper   []byte
	cache    *redis.Client
	cacheTTL time.Duration
}

// APIKeyCacheData stores API key metadata in Redis cache (without the secret)
type APIKeyCacheData struct {
	KeyID      string     `json:"key_id"`
	TenantID   uint       `json:"tenant_id"`
	Revoked    bool       `json:"revoked"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Scopes     []string   `json:"scopes"`
	Salt       string     `json:"salt"`        // base64 encoded
	SecretHash string     `json:"secret_hash"` // base64 encoded
}

func NewAPIKeyService(db *gorm.DB, pepper []byte, cache *redis.Client, ttl time.Duration) *APIKeyService {
	return &APIKeyService{
		db:       db,
		pepper:   pepper,
		cache:    cache,
		cacheTTL: ttl,
	}
}

// CreateKey returns keyID and secret (secret shown once)
func (s *APIKeyService) CreateKey(ctx context.Context, tenantID uint, scopes []string, expiresAt *time.Time) (string, string, error) {
	// generate secret
	sec := make([]byte, 32)
	if _, err := rand.Read(sec); err != nil {
		return "", "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(sec)
	// salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	// hash
	mac := hmac.New(sha256.New, s.pepper)
	mac.Write(salt)
	mac.Write([]byte(secret))
	hash := mac.Sum(nil)

	kid := uuid.New().String()
	ak := models.APIKey{
		TenantID:   tenantID,
		KeyID:      kid,
		Salt:       salt,
		SecretHash: hash,
		Scopes:     scopes,
		Revoked:    false,
		ExpiresAt:  expiresAt,
	}
	if err := s.db.Create(&ak).Error; err != nil {
		return "", "", err
	}
	// cache metadata (not secret)
	if s.cache != nil {
		cacheData := APIKeyCacheData{
			KeyID:      ak.KeyID,
			TenantID:   ak.TenantID,
			Revoked:    ak.Revoked,
			ExpiresAt:  ak.ExpiresAt,
			Scopes:     ak.Scopes,
			Salt:       base64.StdEncoding.EncodeToString(ak.Salt),
			SecretHash: base64.StdEncoding.EncodeToString(ak.SecretHash),
		}
		cacheJSON, err := json.Marshal(cacheData)
		if err == nil {
			cacheKey := "apikey:" + kid
			s.cache.Set(ctx, cacheKey, cacheJSON, s.cacheTTL)
		}
	}
	return kid, secret, nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, presentedKey string) (*models.APIKey, error) {
	// presentedKey expected format: keyid:secret
	var keyID, secret string
	parts := []byte(presentedKey)
	idx := -1
	for i, b := range parts {
		if b == ':' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return nil, errors.New("bad key format")
	}
	keyID = string(parts[:idx])
	secret = string(parts[idx+1:])

	var ak models.APIKey
	cacheKey := "apikey:" + keyID

	// Try cache first
	if s.cache != nil {
		cacheDataStr, err := s.cache.Get(ctx, cacheKey).Result()
		if err == nil {
			// Cache hit - parse metadata
			var cacheData APIKeyCacheData
			if err := json.Unmarshal([]byte(cacheDataStr), &cacheData); err == nil {
				// Quick validation checks from cache
				if cacheData.Revoked {
					return nil, fmt.Errorf("invalid key: key_id %s is revoked", keyID)
				}
				if cacheData.ExpiresAt != nil && cacheData.ExpiresAt.Before(time.Now()) {
					return nil, fmt.Errorf("invalid key: key_id %s expired at %v", keyID, cacheData.ExpiresAt)
				}

				// Decode salt and secret hash from cache
				salt, err := base64.StdEncoding.DecodeString(cacheData.Salt)
				if err != nil {
					// Cache data corrupted, fall through to DB
					goto dbLookup
				}
				secretHash, err := base64.StdEncoding.DecodeString(cacheData.SecretHash)
				if err != nil {
					// Cache data corrupted, fall through to DB
					goto dbLookup
				}

				// Validate secret hash
				mac := hmac.New(sha256.New, s.pepper)
				mac.Write(salt)
				mac.Write([]byte(secret))
				expected := mac.Sum(nil)
				if !hmac.Equal(expected, secretHash) {
					return nil, fmt.Errorf("invalid key: hash mismatch (key_id: %s)", keyID)
				}

				// Validation successful - reconstruct APIKey from cache
				ak = models.APIKey{
					ID:         0, // Not needed for validation
					TenantID:   cacheData.TenantID,
					KeyID:      cacheData.KeyID,
					Salt:       salt,
					SecretHash: secretHash,
					Scopes:     cacheData.Scopes,
					Revoked:    cacheData.Revoked,
					ExpiresAt:  cacheData.ExpiresAt,
				}
				return &ak, nil
			}
		}
		// Cache miss or error - fall through to DB lookup
	}

dbLookup:
	// Cache miss or error - query database
	if err := s.db.Where("key_id = ?", keyID).First(&ak).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid key: key_id %s not found", keyID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Validate from database
	// Check revoked status
	if ak.Revoked {
		return nil, fmt.Errorf("invalid key: key_id %s is revoked", keyID)
	}
	// Check expiration
	if ak.ExpiresAt != nil && ak.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("invalid key: key_id %s expired at %v", keyID, ak.ExpiresAt)
	}

	// Compare hash
	mac := hmac.New(sha256.New, s.pepper)
	mac.Write(ak.Salt)
	mac.Write([]byte(secret))
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, ak.SecretHash) {
		return nil, fmt.Errorf("invalid key: hash mismatch (key_id: %s)", keyID)
	}

	// Update cache with fresh data from DB
	if s.cache != nil {
		cacheData := APIKeyCacheData{
			KeyID:      ak.KeyID,
			TenantID:   ak.TenantID,
			Revoked:    ak.Revoked,
			ExpiresAt:  ak.ExpiresAt,
			Scopes:     ak.Scopes,
			Salt:       base64.StdEncoding.EncodeToString(ak.Salt),
			SecretHash: base64.StdEncoding.EncodeToString(ak.SecretHash),
		}
		cacheJSON, err := json.Marshal(cacheData)
		if err == nil {
			s.cache.Set(ctx, cacheKey, cacheJSON, s.cacheTTL)
		}
	}

	return &ak, nil
}
