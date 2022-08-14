package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	versionEndpoint = "https://hit-server.yolo42.com/api/v1/latest-version"
	requestTimeout  = 3 * time.Second
)

type versionResponse struct {
	Version string `json:"version"`
}

func (c HitClient) LatestHitCLIVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("prepare HTTP request: %w", err)
	}

	req.Header.Add("user-agent", "hit/"+c.hitCLIVersion)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do http request: %w", err)
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			c.logger.Debug("version-check: failed to close response body",
				zap.Error(err))
		}
	}()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %v", res.StatusCode)
	}
	js, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var r versionResponse
	if err := json.Unmarshal(js, &r); err != nil {
		return "", fmt.Errorf("parse HTTP response: %w", err)
	}
	return r.Version, nil
}
