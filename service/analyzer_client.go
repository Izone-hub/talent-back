package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// AnalyzerClient handles direct communication with the Python Talent Analyzer service.
type AnalyzerClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
}

// NewAnalyzerClient creates a shared AnalyzerClient with strict timeout and connection pooling.
func NewAnalyzerClient(baseURL, internalToken string) *AnalyzerClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/analyze-cv")
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &AnalyzerClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient: &http.Client{
			Timeout: 45 * time.Second, // Bounded timeout to prevent hanging connections
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// CVAnalysisResult represents the parsed AI analysis output from the Python analyzer.
type CVAnalysisResult struct {
	RawBody    []byte
	Engine     string
	Strengths  string
	Weaknesses string
}

// AnalyzeCV sends the CV bytes and candidate username directly to the Python Analyzer service.
func (c *AnalyzerClient) AnalyzeCV(ctx context.Context, fileBytes []byte, fileName, githubUsername string) (*CVAnalysisResult, error) {
	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	fw, err := mp.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart file field: %w", err)
	}
	if _, err := fw.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write file bytes: %w", err)
	}

	if githubUsername != "" {
		if err := mp.WriteField("github_username", githubUsername); err != nil {
			return nil, fmt.Errorf("failed to write github_username field: %w", err)
		}
	}
	if err := mp.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	reqURL := c.baseURL + "/analyze-cv"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request to analyzer: %w", err)
	}
	req.Header.Set("Content-Type", mp.FormDataContentType())
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Service-Token", c.internalToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("analyzer service request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read analyzer response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("analyzer returned error status %d: %s", resp.StatusCode, string(body))
	}

	var analysisResp struct {
		Analysis json.RawMessage `json:"analysis"`
		Response json.RawMessage `json:"response"`
		Engine   string          `json:"engine"`
	}
	_ = json.Unmarshal(body, &analysisResp)

	analysisPayload := analysisResp.Analysis
	if len(analysisPayload) == 0 {
		analysisPayload = analysisResp.Response
	}
	if len(analysisPayload) == 0 {
		analysisPayload = body
	}

	strengths, weaknesses := extractStrengthsWeaknessesFromJSON(analysisPayload)

	return &CVAnalysisResult{
		RawBody:    body,
		Engine:     analysisResp.Engine,
		Strengths:  strengths,
		Weaknesses: weaknesses,
	}, nil
}

// extractStrengthsWeaknessesFromJSON parses check items from the AI analysis payload.
func extractStrengthsWeaknessesFromJSON(response json.RawMessage) (string, string) {
	var data struct {
		Checks []struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(response, &data); err != nil {
		return "", ""
	}
	var strengths, weaknesses []string
	for _, c := range data.Checks {
		if c.Status == "pass" {
			strengths = append(strengths, c.Message)
		} else if c.Status == "warn" || c.Status == "fail" {
			weaknesses = append(weaknesses, c.Message)
		}
	}
	return strings.Join(strengths, "; "), strings.Join(weaknesses, "; ")
}
