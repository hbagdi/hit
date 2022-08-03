package model

import (
	"net/http"
	"net/url"
)

type Hit struct {
	ID           int
	HitRequestID string
	CreatedAt    int64
	Request      Request
	// RequestError  RequestError
	// ResponseError ResponseError
	Response Response
	// Latency is NYI.
	Latency Latency
	// Network is NYI.
	Network Network
}

// type RequestError struct {
// 	 Message string
// }
//
// type ResponseError struct {
// 	 Message string
// }

type Request struct {
	Proto       string
	Scheme      string
	Method      string
	Host        string
	Path        string
	QueryString string
	Header      http.Header
	Body        []byte
}

func (r Request) URL() string {
	url := url.URL{
		Scheme:   r.Scheme,
		Host:     r.Host,
		Path:     r.Path,
		RawQuery: r.QueryString,
	}
	return url.String()
}

type Response struct {
	Proto  string
	Code   int
	Status string
	Header http.Header
	Body   []byte
}

type Latency struct {
	DNSResolution int
	TCPConnection int
	TLSConnection int
	// TTFB int ?
}

type Network struct {
	DNSServer string
	IPInUse   string
	PortInUse int
}

type DNSResponse struct {
	// TODO(hbagdi): figure out details.
	RecordType string
	Addresses  []string
}
