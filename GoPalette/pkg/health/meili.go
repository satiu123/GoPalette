package health

import (
	"context"

	"github.com/meilisearch/meilisearch-go"
)

type MeilisearchChecker struct {
	meili meilisearch.ServiceManager
}

func NewMeilisearchChecker(meili meilisearch.ServiceManager) *MeilisearchChecker {
	return &MeilisearchChecker{
		meili: meili,
	}
}

func (c *MeilisearchChecker) Name() string {
	return "meilisearch"
}

func (c *MeilisearchChecker) Check(ctx context.Context) error {
	_, err := c.meili.HealthWithContext(ctx)
	return err
}
