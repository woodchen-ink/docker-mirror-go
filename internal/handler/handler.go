package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/woodchen/docker-mirror-go/internal/backend"
	"github.com/woodchen/docker-mirror-go/internal/token"
)

var (
	proxyHeaderAllowList = []string{"accept", "user-agent", "accept-encoding"}
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

	tokenProvider := token.NewTokenProvider("", "")
	dockerBackend := backend.NewBackend(host, tokenProvider)

	headers := copyProxyHeaders(c.Request.Header)

	log.Info().
		Str("original_path", c.Request.URL.Path).
		Str("rewritten_path", pathname).
		Str("host", host).
		Str("org_name", orgName).
		Msg("Proxying registry request")

	resp, err := dockerBackend.Proxy(pathname, headers)
	if err != nil {
		log.Error().Err(err).Msg("Failed to proxy request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy request"})
		return
	}
	defer resp.Body.Close()

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

	// /v2/repo/manifests/xxx -> /v2/library/repo/manifests/xxx
	// /v2/repo/blobs/xxx -> /v2/library/repo/blobs/xxx
	if orgName == "" && len(splitedPath) == 5 && validActionNames[splitedPath[3]] {
		newPath := []string{splitedPath[0], splitedPath[1], "library", splitedPath[2], splitedPath[3], splitedPath[4]}
		return strings.Join(newPath, "/")
	}

	if orgName == "" || orgNameBackend[orgName] == "" {
		return pathname
	}

	// Remove org name from path for external registries
	var cleanSplitedPath []string
	for i, part := range splitedPath {
		if !(part == orgName && i == 2) {
			cleanSplitedPath = append(cleanSplitedPath, part)
		}
	}
	return strings.Join(cleanSplitedPath, "/")
}
