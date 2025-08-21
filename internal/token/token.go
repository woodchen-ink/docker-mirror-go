package token

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

type WwwAuthenticate struct {
	Realm   string
	Service string
	Scope   string
}

type Token struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

type TokenProvider struct {
	username string
	password string
	cache    *cache.Cache
}

func NewTokenProvider(username, password string) *TokenProvider {
	return &TokenProvider{
		username: username,
		password: password,
		cache:    cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (tp *TokenProvider) GetToken(authenticateStr string) (*Token, error) {
	wwwAuth, err := parseAuthenticateStr(authenticateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse authenticate string: %w", err)
	}

	cacheKey := tp.authenticateCacheKey(wwwAuth)

	// Check cache first
	if cachedToken, found := tp.cache.Get(cacheKey); found {
		if token, ok := cachedToken.(*Token); ok {
			log.Debug().Str("cache_key", cacheKey).Msg("Using cached token")
			return token, nil
		}
	}

	// Fetch new token
	token, err := tp.fetchToken(wwwAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token: %w", err)
	}

	// Cache the token
	expiration := time.Duration(token.ExpiresIn) * time.Second
	tp.cache.Set(cacheKey, token, expiration)

	log.Info().Str("cache_key", cacheKey).Int("expires_in", token.ExpiresIn).Msg("Cached new token")

	return token, nil
}

func parseAuthenticateStr(authenticateStr string) (*WwwAuthenticate, error) {
	parts := strings.SplitN(authenticateStr, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, fmt.Errorf("invalid Www-Authenticate header: %s", authenticateStr)
	}

	params := parts[1]
	auth := &WwwAuthenticate{}

	// Parse realm
	if realm := extractParam(params, "realm"); realm != "" {
		auth.Realm = realm
	}

	// Parse service
	if service := extractParam(params, "service"); service != "" {
		auth.Service = service
	}

	// Parse scope
	if scope := extractParam(params, "scope"); scope != "" {
		auth.Scope = scope
	}

	return auth, nil
}

func extractParam(params, name string) string {
	re := regexp.MustCompile(name + `="([^"]*)"`)
	matches := re.FindStringSubmatch(params)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (tp *TokenProvider) authenticateCacheKey(wwwAuth *WwwAuthenticate) string {
	keyStr := fmt.Sprintf("%s:%s/%s/%s/%s", tp.username, tp.password, wwwAuth.Realm, wwwAuth.Service, wwwAuth.Scope)
	hash := sha256.Sum256([]byte(keyStr))
	return fmt.Sprintf("token/%s", hex.EncodeToString(hash[:]))
}

func (tp *TokenProvider) fetchToken(wwwAuth *WwwAuthenticate) (*Token, error) {
	authURL, err := url.Parse(wwwAuth.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse realm URL: %w", err)
	}

	// Add query parameters
	query := authURL.Query()
	if wwwAuth.Service != "" {
		query.Set("service", wwwAuth.Service)
	}
	if wwwAuth.Scope != "" {
		query.Set("scope", wwwAuth.Scope)
	}
	authURL.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", authURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	// TODO: Add basic auth support if username/password provided
	// if tp.username != "" && tp.password != "" {
	//     req.SetBasicAuth(tp.username, tp.password)
	// }

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token from %s: %w", authURL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp Token
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}