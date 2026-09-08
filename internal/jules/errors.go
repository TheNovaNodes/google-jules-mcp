package jules

import (
	"encoding/json"
	"fmt"
)

// APIError represents an error returned by the Google Jules REST API.
type APIError struct {
	StatusCode int
	RawBody    string
	Message    string
	Attempts   int
}

type googleErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Error formats the APIError into an actionable error message matching spec R4.
func (e *APIError) Error() string {
	switch {
	case e.StatusCode == 404:
		return "Session or source not found — check session_id or source_name (use list_jules_sources)"
	case e.StatusCode == 401 || e.StatusCode == 403:
		return "JULES_API_KEY is invalid or lacks required permissions"
	case e.StatusCode == 400:
		// Attempt to extract Google's reason string
		var gErr googleErrorResponse
		if err := json.Unmarshal([]byte(e.RawBody), &gErr); err == nil && gErr.Error.Message != "" {
			return fmt.Sprintf("Invalid argument: %s", gErr.Error.Message)
		}
		if e.Message != "" {
			return fmt.Sprintf("Invalid argument: %s", e.Message)
		}
		return "Invalid argument provided to Jules API"
	case e.StatusCode == 429:
		if e.Attempts > 0 {
			return fmt.Sprintf("Rate-limited by Google Jules API after %d attempts; retry after backoff", e.Attempts)
		}
		return "Rate-limited by Google Jules API; retry after backoff"
	case e.StatusCode >= 500:
		if e.Attempts > 0 {
			return fmt.Sprintf("Google Jules API unavailable (status %d); attempted %d times", e.StatusCode, e.Attempts)
		}
		return fmt.Sprintf("Google Jules API unavailable (status %d)", e.StatusCode)
	default:
		if e.Message != "" {
			return fmt.Sprintf("Jules API error (status %d): %s", e.StatusCode, e.Message)
		}
		return fmt.Sprintf("Jules API error (status %d): %s", e.StatusCode, e.RawBody)
	}
}
