package client

import (
	"net/http"

	"go.uber.org/zap"
)

type HitClient struct {
	httpClient    *http.Client
	hitCLIVersion string

	logger *zap.Logger
}

type HitClientOpts struct {
	Logger        *zap.Logger
	HitCLIVersion string
}

func NewHitClient(opts HitClientOpts) (*HitClient, error) {
	c := &HitClient{}
	c.httpClient = &http.Client{}
	c.logger = opts.Logger
	c.hitCLIVersion = opts.HitCLIVersion

	return c, nil
}
