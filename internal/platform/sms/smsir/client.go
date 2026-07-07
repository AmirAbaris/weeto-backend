package smsir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Parameter struct {
	Name  string
	Value string
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type verifyRequest struct {
	Mobile     string            `json:"mobile"`
	TemplateID int               `json:"templateId"`
	Parameters []verifyParameter `json:"parameters"`
}

type verifyParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type apiResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    *verifyResponse `json:"data"`
}

type verifyResponse struct {
	MessageID int64   `json:"messageId"`
	Cost      float64 `json:"cost"`
}

func (c *Client) VerifySend(ctx context.Context, mobile string, templateID int, params []Parameter) (int64, float64, error) {
	reqParams := make([]verifyParameter, len(params))
	for i, p := range params {
		reqParams[i] = verifyParameter{Name: p.Name, Value: p.Value}
	}

	body, err := json.Marshal(verifyRequest{
		Mobile:     mobile,
		TemplateID: templateID,
		Parameters: reqParams,
	})
	if err != nil {
		return 0, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/send/verify", bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrAPIFailure, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: read body: %v", ErrAPIFailure, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, 0, fmt.Errorf("%w: http %d: %s", ErrAPIFailure, resp.StatusCode, string(respBody))
	}

	var parsed apiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return 0, 0, fmt.Errorf("%w: decode response: %v", ErrAPIFailure, err)
	}
	if parsed.Status != 1 || parsed.Data == nil {
		return 0, 0, fmt.Errorf("%w: %s", ErrRejected, parsed.Message)
	}

	return parsed.Data.MessageID, parsed.Data.Cost, nil
}
