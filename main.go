package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// StatusResponse represents the Gitea commit status response
type StatusResponse struct {
	State      string `json:"state"`
	Statuses   []any  `json:"statuses"`
	TotalCount int    `json:"total_count"`
}

// Repository represents basic repo info from Gitea
type Repository struct {
	DefaultBranch string `json:"default_branch"`
}

// BuildStatusResponse represents our API response
type BuildStatusResponse struct {
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	State      string `json:"state"`
	Symbol     string `json:"symbol"`
	Error      string `json:"error,omitempty"`
}

// GiteaService handles interactions with Gitea API
type GiteaService struct {
	BaseURL    string
	Token      string
	HTTPClient HTTPClient
}

// HTTPClient interface for testing
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	giteaURL string
	token    string
	client   *http.Client
	service  *GiteaService
)

func init() {
	giteaURL = os.Getenv("GITEA_URL")
	if giteaURL == "" {
		log.Fatal("GITEA_URL environment variable is required")
	}

	token = os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN environment variable is required")
	}

	// Create HTTP client with timeout
	client = &http.Client{
		Timeout: 10 * time.Second,
	}

	// Initialize service
	service = &GiteaService{
		BaseURL:    giteaURL,
		Token:      token,
		HTTPClient: client,
	}
}

// GetDefaultBranch fetches the default branch for a repository
func (g *GiteaService) GetDefaultBranch(owner, repo string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s", g.BaseURL, owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.Token))

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get repository info: %d - %s", resp.StatusCode, string(body))
	}

	var repository Repository
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return "", err
	}

	return repository.DefaultBranch, nil
}

// getDefaultBranch is a wrapper for backward compatibility
func getDefaultBranch(owner, repo string) (string, error) {
	return service.GetDefaultBranch(owner, repo)
}

// GetCommitStatus fetches the commit status for a repository
func (g *GiteaService) GetCommitStatus(owner, repo, branch string) (*StatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/commits/%s/status", g.BaseURL, owner, repo, branch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.Token))

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		// No status available
		return &StatusResponse{State: "unknown"}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get commit status: %d - %s", resp.StatusCode, string(body))
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// getCommitStatus is a wrapper for backward compatibility
func getCommitStatus(owner, repo, branch string) (*StatusResponse, error) {
	return service.GetCommitStatus(owner, repo, branch)
}

// mapStateToSymbol converts Gitea state to a symbol
func mapStateToSymbol(state string) string {
	symbolMap := map[string]string{
		"success": "✓",
		"failure": "✗",
		"error":   "✗",
		"pending": "●",
		"warning": "⚠",
		"unknown": "○",
	}

	if symbol, ok := symbolMap[state]; ok {
		return symbol
	}
	return "?"
}

// mapStateToHTTPCode converts Gitea state to appropriate HTTP status code
func mapStateToHTTPCode(state string) int {
	codeMap := map[string]int{
		"success": http.StatusOK,                  // 200
		"failure": http.StatusExpectationFailed,   // 417
		"error":   http.StatusInternalServerError, // 500
		"pending": http.StatusAccepted,            // 202
		"warning": http.StatusOK,                  // 200 (successful but with warnings)
		"unknown": http.StatusNoContent,           // 204
	}

	if code, ok := codeMap[state]; ok {
		return code
	}
	return http.StatusOK // default to 200
}

// statusHandler handles the /status endpoint
func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")

	if owner == "" || repo == "" {
		response := BuildStatusResponse{
			Error: "Both 'owner' and 'repo' query parameters are required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	// Get default branch
	branch, err := getDefaultBranch(owner, repo)
	if err != nil {
		response := BuildStatusResponse{
			Owner:      owner,
			Repository: repo,
			Error:      fmt.Sprintf("Failed to get repository info: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	// Get commit status
	status, err := getCommitStatus(owner, repo, branch)
	if err != nil {
		response := BuildStatusResponse{
			Owner:      owner,
			Repository: repo,
			Branch:     branch,
			Error:      fmt.Sprintf("Failed to get commit status: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	// Build response
	response := BuildStatusResponse{
		Owner:      owner,
		Repository: repo,
		Branch:     branch,
		State:      status.State,
		Symbol:     mapStateToSymbol(status.State),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(mapStateToHTTPCode(status.State))
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/health", healthHandler)

	// Log middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	log.Printf("Gitea URL: %s", giteaURL)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
