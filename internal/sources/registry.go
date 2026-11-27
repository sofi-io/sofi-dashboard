package sources

import (
	"fmt"
	"strings"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
	"github.com/sofi-io/sofi-dashboard/internal/sources/cbf"
	"github.com/sofi-io/sofi-dashboard/internal/sources/fpf"
	"github.com/sofi-io/sofi-dashboard/internal/sources/httpclient"
)

func NewDiscoverer(source string) (canonical.Discoverer, error) {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "cbf":
		client, err := httpclient.NewClient(httpclient.ClientConfig{})
		if err != nil {
			return nil, fmt.Errorf("initialize CBF client: %w", err)
		}

		return cbf.NewAdapter(client), nil

	case "fpf":
		client, err := fpf.NewClient()
		if err != nil {
			return nil, fmt.Errorf("initialize FPF client: %w", err)
		}

		return fpf.NewAdapter(client), nil

	default:
		return nil, fmt.Errorf(
			"unsupported discovery source: %q",
			source,
		)
	}
}
