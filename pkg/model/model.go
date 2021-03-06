package model

import "net/http"

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
	Method      string
	Host        string
	Path        string
	QueryString string
	Header      http.Header
	Body        []byte
}

type Response struct {
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
