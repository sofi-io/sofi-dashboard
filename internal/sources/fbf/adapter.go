package fbf

import (
	"context"
	"fmt"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
	"github.com/sofi-io/sofi-dashboard/internal/sources/httpclient"
)

var _ canonical.Discoverer = (*Adapter)(nil)

type Adapter struct {
	client *httpclient.Client
}

func NewAdapter(client *httpclient.Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

func (a *Adapter) DiscoverCompetition(
	ctx context.Context,
	competitionURL string,
	options canonical.DiscoveryOptions,
) (*canonical.Result, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("FBF adapter client is not initialized")
	}

	return &canonical.Result{}, nil
}
