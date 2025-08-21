package backend

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/woodchen/docker-mirror-go/internal/token"
)

type Backend struct {
	host          string
	tokenProvider *token.TokenProvider
}

func NewBackend(host string, tokenProvider *token.TokenProvider) *Backend {
	return &Backend{
		host:          host,
		tokenProvider: tokenProvider,
	}
}

func (b *Backend) Proxy(pathname string, headers http.Header) (*http.Response, error) {
	targetURL, err := url.Parse(b.host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host URL: %w", err)
	}

	targetURL.Path = pathname

	req, err := http.NewRequest("GET", targetURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for k, v := range headers {
		req.Header[k] = v
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// If no token provider or response is not 401, return as is
	if b.tokenProvider == nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Handle authentication
	authenticateStr := resp.Header.Get("Www-Authenticate")
	if authenticateStr == "" {
		return resp, nil
	}

	// Close the unauthorized response
	resp.Body.Close()

	log.Info().Str("authenticate", authenticateStr).Msg("Handling authentication")

	authToken, err := b.tokenProvider.GetToken(authenticateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Create authenticated request
	authReq, err := http.NewRequest("GET", targetURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated request: %w", err)
	}

	// Copy original headers
	for k, v := range headers {
		authReq.Header[k] = v
	}

	// Add authorization header
	authReq.Header.Set("Authorization", "Bearer "+authToken.Token)

	// Execute authenticated request
	authResp, err := client.Do(authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute authenticated request: %w", err)
	}

	return authResp, nil
}
