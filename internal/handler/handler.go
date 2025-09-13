package handler

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/woodchen/docker-mirror-go/internal/backend"
	"github.com/woodchen/docker-mirror-go/internal/token"
)

var (
	proxyHeaderAllowList = []string{"accept", "user-agent", "accept-encoding", "authorization"}
	validActionNames     = map[string]bool{
		"manifests": true,
		"blobs":     true,
		"tags":      true,
		"referrers": true,
	}
	orgNameBackend = map[string]string{
		"gcr":    "https://gcr.io",
		"k8sgcr": "https://k8s.gcr.io",
		"quay":   "https://quay.io",
		"ghcr":   "https://ghcr.io",
	}
	defaultBackendHost = "https://registry-1.docker.io"
)

func HandleRegistryRequest(c *gin.Context) {
	orgName := orgNameFromPath(c.Request.URL.Path)
	pathname := rewritePath(orgName, c.Request.URL.Path)
	host := hostByOrgName(orgName)

	// Extract credentials from Authorization header
	username, password := getCredentialsFromRequest(c)
	tokenProvider := token.NewTokenProvider(username, password)
	dockerBackend := backend.NewBackend(host, tokenProvider)

	headers := copyProxyHeaders(c.Request.Header)

	log.Info().
		Str("method", c.Request.Method).
		Str("original_path", c.Request.URL.Path).
		Str("rewritten_path", pathname).
		Str("host", host).
		Str("org_name", orgName).
		Str("username", username).
		Bool("has_credentials", username != "" && password != "").
		Msg("Proxying registry request")

	resp, err := dockerBackend.Proxy(c.Request.Method, pathname, headers)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", c.Request.Method).
			Str("path", pathname).
			Str("host", host).
			Msg("Failed to proxy request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy request"})
		return
	}
	defer resp.Body.Close()

	log.Info().
		Str("method", c.Request.Method).
		Str("path", pathname).
		Int("status", resp.StatusCode).
		Msg("Proxy response received")

	// Copy response headers
	for k, v := range resp.Header {
		for _, val := range v {
			c.Header(k, val)
		}
	}

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

func copyProxyHeaders(inputHeaders http.Header) http.Header {
	headers := http.Header{}
	for _, headerName := range proxyHeaderAllowList {
		if values := inputHeaders[http.CanonicalHeaderKey(headerName)]; len(values) > 0 {
			headers[http.CanonicalHeaderKey(headerName)] = values
		}
	}
	return headers
}

// getCredentialsFromRequest extracts credentials from Authorization header
func getCredentialsFromRequest(c *gin.Context) (string, string) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
		// Decode Basic Auth
		encoded := strings.TrimPrefix(authHeader, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				log.Info().
					Str("username", parts[0]).
					Msg("Using credentials from Authorization header")
				return parts[0], parts[1]
			}
		}
	}

	// No credentials provided
	log.Info().Msg("No Authorization header found")
	return "", ""
}

func orgNameFromPath(pathname string) string {
	splitedPath := strings.Split(pathname, "/")
	if len(splitedPath) >= 3 && splitedPath[1] == "v2" && splitedPath[2] != "" {
		return strings.ToLower(splitedPath[2])
	}
	return ""
}

func hostByOrgName(orgName string) string {
	if host, exists := orgNameBackend[orgName]; exists {
		return host
	}
	return defaultBackendHost
}

func rewritePath(orgName, pathname string) string {
	splitedPath := strings.Split(pathname, "/")

	// For official Docker Hub images (not in orgNameBackend), add library/ prefix
	// /v2/mysql/manifests/xxx -> /v2/library/mysql/manifests/xxx
	// /v2/nginx/blobs/xxx -> /v2/library/nginx/blobs/xxx
	if orgName != "" && orgNameBackend[orgName] == "" && len(splitedPath) == 5 && validActionNames[splitedPath[3]] {
		newPath := []string{splitedPath[0], splitedPath[1], "library", splitedPath[2], splitedPath[3], splitedPath[4]}
		return strings.Join(newPath, "/")
	}

	// For /v2/ requests without org name
	if orgName == "" {
		return pathname
	}

	// For external registries (gcr, quay, etc.), remove org name from path
	if orgNameBackend[orgName] != "" {
		var cleanSplitedPath []string
		for i, part := range splitedPath {
			if !(part == orgName && i == 2) {
				cleanSplitedPath = append(cleanSplitedPath, part)
			}
		}
		return strings.Join(cleanSplitedPath, "/")
	}

	return pathname
}
