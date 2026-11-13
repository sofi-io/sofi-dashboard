package sofi

import (
	"encoding/json"
	"time"
)

type Extraction struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ExtractionStatus struct {
	ID               string          `json:"id"`
	Status           string          `json:"status"`
	OriginalFilename string          `json:"original_filename"`
	CreatedAt        time.Time       `json:"created_at"`
	CompletedAt      *time.Time      `json:"completed_at"`
	ErrorCode        *string         `json:"error_code,omitempty"`
	Result           json.RawMessage `json:"result"`
	Validation       json.RawMessage `json:"validation"`
}

type APIError struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}
