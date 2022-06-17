package cache

import "net/http"

type Hit struct {
	ID           int
	HitRequestID string
	CreatedAt    int64
	Request      Request
	// RequestError  RequestError
	// ResponseError ResponseError
	Response Response
	Latency  Latency
	Network  Network
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
	Headers     http.Header
	Body        []byte
}

type Response struct {
	Code    int
	Headers http.Header
	Body    []byte
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
