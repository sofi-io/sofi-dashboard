package httpclient

import (
	"net/http"
	"time"
)

type ClientConfig struct {
	Timeout         time.Duration
	MaxResponseSize int64
	UserAgent       string
	Headers         http.Header
}

type Client struct {
	httpClient      *http.Client
	maxResponseSize int64
	headers         http.Header
}
