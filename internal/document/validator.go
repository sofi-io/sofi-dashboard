package document

import (
	"bufio"
	"fmt"
	"mime"
	"strings"
)

func validateContentType(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("document content type is missing")
	}

	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return "", fmt.Errorf("invalid content type: %w", err)
	}

	mediaType = strings.ToLower(mediaType)

	switch mediaType {
	case "application/pdf":
		return mediaType, nil

	case "application/octet-stream":
		// Some official federation servers return
		// PDFs as generic binary content.
		return "application/pdf", nil

	default:
		return "", fmt.Errorf(
			"unsupported document content type: %s",
			mediaType,
		)
	}
}

func validatePDF(reader *bufio.Reader) error {
	header, err := reader.Peek(5)
	if err != nil {
		return fmt.Errorf("read PDF header: %w", err)
	}

	if string(header) != "%PDF-" {
		return fmt.Errorf("document is not a valid PDF")
	}

	return nil
}
