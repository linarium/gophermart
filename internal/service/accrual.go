package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AccrualClient struct {
	baseURL string
	client  *http.Client
}

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"` // REGISTERED, INVALID, PROCESSING, PROCESSED
	Accrual float64 `json:"accrual,omitempty"`
}

func NewAccrualClient(baseURL string) *AccrualClient {
	return &AccrualClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *AccrualClient) GetOrder(ctx context.Context, number string) (*AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var res AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &res, nil
	case http.StatusNoContent:
		return nil, errors.New("order not registered")
	case http.StatusTooManyRequests:
		return nil, errors.New("rate limit exceeded")
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}
}
