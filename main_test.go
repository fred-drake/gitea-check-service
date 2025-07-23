package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockHTTPClient implements HTTPClient interface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// Helper function to create HTTP response
func createHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestMapStateToSymbol(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"success", "✓"},
		{"failure", "✗"},
		{"error", "✗"},
		{"pending", "●"},
		{"warning", "⚠"},
		{"unknown", "○"},
		{"invalid", "?"},
		{"", "?"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("state_%s", tt.state), func(t *testing.T) {
			result := mapStateToSymbol(tt.state)
			if result != tt.expected {
				t.Errorf("mapStateToSymbol(%s) = %s, want %s", tt.state, result, tt.expected)
			}
		})
	}
}

func TestMapStateToHTTPCode(t *testing.T) {
	tests := []struct {
		state    string
		expected int
	}{
		{"success", http.StatusOK},
		{"failure", http.StatusExpectationFailed},
		{"error", http.StatusInternalServerError},
		{"pending", http.StatusAccepted},
		{"warning", http.StatusOK},
		{"unknown", http.StatusNoContent},
		{"invalid", http.StatusOK},
		{"", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("state_%s", tt.state), func(t *testing.T) {
			result := mapStateToHTTPCode(tt.state)
			if result != tt.expected {
				t.Errorf("mapStateToHTTPCode(%s) = %d, want %d", tt.state, result, tt.expected)
			}
		})
	}
}

func TestGiteaService_GetDefaultBranch(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		mockResponse   *http.Response
		mockError      error
		expectedBranch string
		expectedError  string
	}{
		{
			name:  "successful request",
			owner: "testowner",
			repo:  "testrepo",
			mockResponse: createHTTPResponse(200, `{
                "default_branch": "main",
                "name": "testrepo"
            }`),
			mockError:      nil,
			expectedBranch: "main",
			expectedError:  "",
		},
		{
			name:  "repository not found",
			owner: "testowner",
			repo:  "nonexistent",
			mockResponse: createHTTPResponse(404, `{
                "message": "Repository not found"
            }`),
			mockError:     nil,
			expectedError: "failed to get repository info: 404",
		},
		{
			name:          "network error",
			owner:         "testowner",
			repo:          "testrepo",
			mockError:     fmt.Errorf("network error"),
			expectedError: "network error",
		},
		{
			name:  "invalid JSON response",
			owner: "testowner",
			repo:  "testrepo",
			mockResponse: createHTTPResponse(200, `{
                "invalid": "json",
            }`),
			mockError:     nil,
			expectedError: "invalid character",
		},
		{
			name:  "unauthorized access",
			owner: "testowner",
			repo:  "privaterepo",
			mockResponse: createHTTPResponse(401, `{
                "message": "Unauthorized"
            }`),
			mockError:     nil,
			expectedError: "failed to get repository info: 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					// Verify request URL and headers
					expectedURL := fmt.Sprintf("https://git.example.com/api/v1/repos/%s/%s", tt.owner, tt.repo)
					if req.URL.String() != expectedURL {
						t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
					}

					authHeader := req.Header.Get("Authorization")
					if authHeader != "token test-token" {
						t.Errorf("Expected Authorization header 'token test-token', got '%s'", authHeader)
					}

					return tt.mockResponse, tt.mockError
				},
			}

			service := &GiteaService{
				BaseURL:    "https://git.example.com",
				Token:      "test-token",
				HTTPClient: mockClient,
			}

			branch, err := service.GetDefaultBranch(tt.owner, tt.repo)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if branch != tt.expectedBranch {
					t.Errorf("Expected branch '%s', got '%s'", tt.expectedBranch, branch)
				}
			}
		})
	}
}

func TestGiteaService_GetCommitStatus(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		branch         string
		mockResponse   *http.Response
		mockError      error
		expectedStatus *StatusResponse
		expectedError  string
	}{
		{
			name:   "successful status request",
			owner:  "testowner",
			repo:   "testrepo",
			branch: "main",
			mockResponse: createHTTPResponse(200, `{
                "state": "success",
                "statuses": [],
                "total_count": 1
            }`),
			mockError: nil,
			expectedStatus: &StatusResponse{
				State:      "success",
				Statuses:   []any{},
				TotalCount: 1,
			},
			expectedError: "",
		},
		{
			name:   "no status found - returns unknown",
			owner:  "testowner",
			repo:   "testrepo",
			branch: "main",
			mockResponse: createHTTPResponse(404, `{
                "message": "Not Found"
            }`),
			mockError: nil,
			expectedStatus: &StatusResponse{
				State: "unknown",
			},
			expectedError: "",
		},
		{
			name:   "pending status",
			owner:  "testowner",
			repo:   "testrepo",
			branch: "develop",
			mockResponse: createHTTPResponse(200, `{
                "state": "pending",
                "statuses": [{"state": "pending", "context": "ci/test"}],
                "total_count": 1
            }`),
			mockError: nil,
			expectedStatus: &StatusResponse{
				State:      "pending",
				Statuses:   []any{map[string]any{"state": "pending", "context": "ci/test"}},
				TotalCount: 1,
			},
			expectedError: "",
		},
		{
			name:          "network error",
			owner:         "testowner",
			repo:          "testrepo",
			branch:        "main",
			mockError:     fmt.Errorf("connection timeout"),
			expectedError: "connection timeout",
		},
		{
			name:   "server error",
			owner:  "testowner",
			repo:   "testrepo",
			branch: "main",
			mockResponse: createHTTPResponse(500, `{
                "message": "Internal Server Error"
            }`),
			mockError:     nil,
			expectedError: "failed to get commit status: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					// Verify request URL and headers
					expectedURL := fmt.Sprintf("https://git.example.com/api/v1/repos/%s/%s/commits/%s/status", tt.owner, tt.repo, tt.branch)
					if req.URL.String() != expectedURL {
						t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
					}

					authHeader := req.Header.Get("Authorization")
					if authHeader != "token test-token" {
						t.Errorf("Expected Authorization header 'token test-token', got '%s'", authHeader)
					}

					return tt.mockResponse, tt.mockError
				},
			}

			service := &GiteaService{
				BaseURL:    "https://git.example.com",
				Token:      "test-token",
				HTTPClient: mockClient,
			}

			status, err := service.GetCommitStatus(tt.owner, tt.repo, tt.branch)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if status.State != tt.expectedStatus.State {
					t.Errorf("Expected state '%s', got '%s'", tt.expectedStatus.State, status.State)
				}
				if status.TotalCount != tt.expectedStatus.TotalCount {
					t.Errorf("Expected total_count %d, got %d", tt.expectedStatus.TotalCount, status.TotalCount)
				}
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"status":"ok"}`
	body := strings.TrimSpace(rr.Body.String())
	if body != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			body, expected)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}
}

func TestStatusHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		queryParams        string
		mockRepoResponse   *http.Response
		mockRepoError      error
		mockStatusResponse *http.Response
		mockStatusError    error
		expectedStatus     int
		expectedResponse   BuildStatusResponse
		checkResponse      bool
	}{
		{
			name:        "successful request",
			method:      "GET",
			queryParams: "owner=testowner&repo=testrepo",
			mockRepoResponse: createHTTPResponse(200, `{
                "default_branch": "main"
            }`),
			mockStatusResponse: createHTTPResponse(200, `{
                "state": "success",
                "statuses": [],
                "total_count": 1
            }`),
			expectedStatus: http.StatusOK,
			expectedResponse: BuildStatusResponse{
				Owner:      "testowner",
				Repository: "testrepo",
				Branch:     "main",
				State:      "success",
				Symbol:     "✓",
			},
			checkResponse: true,
		},
		{
			name:           "missing query parameters",
			method:         "GET",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedResponse: BuildStatusResponse{
				Error: "Both 'owner' and 'repo' query parameters are required",
			},
			checkResponse: true,
		},
		{
			name:           "missing repo parameter",
			method:         "GET",
			queryParams:    "owner=testowner",
			expectedStatus: http.StatusBadRequest,
			expectedResponse: BuildStatusResponse{
				Error: "Both 'owner' and 'repo' query parameters are required",
			},
			checkResponse: true,
		},
		{
			name:           "wrong method",
			method:         "POST",
			queryParams:    "owner=testowner&repo=testrepo",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
		{
			name:        "repository not found",
			method:      "GET",
			queryParams: "owner=testowner&repo=nonexistent",
			mockRepoResponse: createHTTPResponse(404, `{
                "message": "Repository not found"
            }`),
			mockRepoError:  nil,
			expectedStatus: http.StatusInternalServerError,
			expectedResponse: BuildStatusResponse{
				Owner:      "testowner",
				Repository: "nonexistent",
				Error:      "Failed to get repository info:",
			},
			checkResponse: true,
		},
		{
			name:        "failure status",
			method:      "GET",
			queryParams: "owner=testowner&repo=testrepo",
			mockRepoResponse: createHTTPResponse(200, `{
                "default_branch": "main"
            }`),
			mockStatusResponse: createHTTPResponse(200, `{
                "state": "failure",
                "statuses": [],
                "total_count": 1
            }`),
			expectedStatus: http.StatusExpectationFailed,
			expectedResponse: BuildStatusResponse{
				Owner:      "testowner",
				Repository: "testrepo",
				Branch:     "main",
				State:      "failure",
				Symbol:     "✗",
			},
			checkResponse: true,
		},
		{
			name:        "pending status",
			method:      "GET",
			queryParams: "owner=testowner&repo=testrepo",
			mockRepoResponse: createHTTPResponse(200, `{
                "default_branch": "develop"
            }`),
			mockStatusResponse: createHTTPResponse(200, `{
                "state": "pending",
                "statuses": [],
                "total_count": 1
            }`),
			expectedStatus: http.StatusAccepted,
			expectedResponse: BuildStatusResponse{
				Owner:      "testowner",
				Repository: "testrepo",
				Branch:     "develop",
				State:      "pending",
				Symbol:     "●",
			},
			checkResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			callCount := 0
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					callCount++
					if callCount == 1 {
						// First call is for repository info
						return tt.mockRepoResponse, tt.mockRepoError
					}
					// Second call is for commit status
					return tt.mockStatusResponse, tt.mockStatusError
				},
			}

			// Temporarily replace the global service for testing
			originalService := service
			service = &GiteaService{
				BaseURL:    "https://git.example.com",
				Token:      "test-token",
				HTTPClient: mockClient,
			}
			defer func() { service = originalService }()

			url := "/status"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req, err := http.NewRequest(tt.method, url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(statusHandler)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.checkResponse {
				contentType := rr.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("handler returned wrong content type: got %v want %v",
						contentType, "application/json")
				}

				var response BuildStatusResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Could not parse response JSON: %v", err)
				}

				if response.Owner != tt.expectedResponse.Owner {
					t.Errorf("Expected owner '%s', got '%s'", tt.expectedResponse.Owner, response.Owner)
				}
				if response.Repository != tt.expectedResponse.Repository {
					t.Errorf("Expected repository '%s', got '%s'", tt.expectedResponse.Repository, response.Repository)
				}
				if response.Branch != tt.expectedResponse.Branch {
					t.Errorf("Expected branch '%s', got '%s'", tt.expectedResponse.Branch, response.Branch)
				}
				if response.State != tt.expectedResponse.State {
					t.Errorf("Expected state '%s', got '%s'", tt.expectedResponse.State, response.State)
				}
				if response.Symbol != tt.expectedResponse.Symbol {
					t.Errorf("Expected symbol '%s', got '%s'", tt.expectedResponse.Symbol, response.Symbol)
				}
				if tt.expectedResponse.Error != "" && !strings.Contains(response.Error, tt.expectedResponse.Error) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedResponse.Error, response.Error)
				}
			}
		})
	}
}

func TestStatusHandler_IntegrationFlow(t *testing.T) {
	// Test the complete flow with multiple HTTP calls
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			url := req.URL.String()

			if strings.Contains(url, "/repos/testowner/testrepo") && !strings.Contains(url, "/commits/") {
				// Repository info request
				return createHTTPResponse(200, `{
                    "default_branch": "main",
                    "name": "testrepo"
                }`), nil
			} else if strings.Contains(url, "/commits/main/status") {
				// Commit status request
				return createHTTPResponse(200, `{
                    "state": "success",
                    "statuses": [{"state": "success", "context": "ci/test"}],
                    "total_count": 1
                }`), nil
			}

			return createHTTPResponse(404, "Not found"), nil
		},
	}

	// Replace global service
	originalService := service
	service = &GiteaService{
		BaseURL:    "https://git.example.com",
		Token:      "test-token",
		HTTPClient: mockClient,
	}
	defer func() { service = originalService }()

	req, err := http.NewRequest("GET", "/status?owner=testowner&repo=testrepo", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statusHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response BuildStatusResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse response JSON: %v", err)
	}

	expected := BuildStatusResponse{
		Owner:      "testowner",
		Repository: "testrepo",
		Branch:     "main",
		State:      "success",
		Symbol:     "✓",
	}

	if response != expected {
		t.Errorf("Expected response %+v, got %+v", expected, response)
	}
}
