package logging

import (
	"net/http"
	"net/url"
)

func RedactedValue() string {
	return "[REDACTED]"
}

func TokenPresent(token string) bool {
	return token != ""
}

func SanitizeURL(raw string) (host string, path string) {
	if raw == "" {
		return "", ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	return parsed.Host, parsed.EscapedPath()
}

func SanitizeURLRequestURL(rawURL *url.URL) (host string, path string) {
	if rawURL == nil {
		return "", ""
	}
	return rawURL.Host, rawURL.EscapedPath()
}

func SanitizeURLRequest(req *http.Request) (host string, path string) {
	if req == nil {
		return "", ""
	}
	return SanitizeURLRequestURL(req.URL)
}
