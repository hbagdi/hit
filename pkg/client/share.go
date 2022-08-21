package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type ShareAPIRequest struct {
	Data ShareData `json:"data,omitempty"`
}

type ShareData struct {
	Request  ShareRequest  `json:"request"`
	Response ShareResponse `json:"response"`
}

type ShareRequest struct {
	Proto       string      `json:"proto,omitempty"`
	Scheme      string      `json:"scheme,omitempty"`
	Method      string      `json:"method,omitempty"`
	Host        string      `json:"host,omitempty"`
	Path        string      `json:"path,omitempty"`
	QueryString string      `json:"query_string,omitempty"`
	Header      http.Header `json:"header,omitempty"`
	// Body contains the HTTP request body encoded with base64.
	Body string `json:"body,omitempty"`
}

type ShareResponse struct {
	Proto  string      `json:"proto,omitempty"`
	Code   int         `json:"code,omitempty"`
	Status string      `json:"status,omitempty"`
	Header http.Header `json:"header,omitempty"`
	// Body contains the HTTP response body encoded with base64.
	Body string `json:"body,omitempty"`
}

type ShareAPIResponse struct {
	ID        string    `json:"id,omitempty"`
	CreatedAt int64     `json:"created_at,omitempty"`
	Data      ShareData `json:"data,omitempty"`
}

const (
	shareEndpoint = "/hits"
)

func (c HitClient) ShareHit(ctx context.Context, request ShareAPIRequest) (ShareAPIResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	requestBody, err := json.Marshal(request)
	if err != nil {
		return ShareAPIResponse{}, fmt.Errorf("marshal request into json: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+shareEndpoint,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return ShareAPIResponse{}, fmt.Errorf("prepare HTTP request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "hit/"+c.hitCLIVersion)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ShareAPIResponse{}, fmt.Errorf("do http request: %w", err)
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			c.logger.Debug("share hit: failed to close response body",
				zap.Error(err))
		}
	}()
	switch res.StatusCode {
	case http.StatusCreated:
		responseBody, err := io.ReadAll(res.Body)
		if err != nil {
			return ShareAPIResponse{}, err
		}
		var r ShareAPIResponse
		if err := json.Unmarshal(responseBody, &r); err != nil {
			return ShareAPIResponse{}, fmt.Errorf("parse HTTP response: %w", err)
		}
		return r, nil
	case http.StatusBadRequest:
		responseBody, err := io.ReadAll(res.Body)
		if err != nil {
			return ShareAPIResponse{}, err
		}
		var m map[string]string
		if err := json.Unmarshal(responseBody, &m); err != nil {
			return ShareAPIResponse{}, fmt.Errorf("parse HTTP response: %w", err)
		}
		return ShareAPIResponse{}, fmt.Errorf("%s", m["message"])
	default:

		return ShareAPIResponse{}, fmt.Errorf("unexpected status code: %v",
			res.StatusCode)
	}
}
