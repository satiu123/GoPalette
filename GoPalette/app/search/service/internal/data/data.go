package data

import (
	"context"
	"errors"

	"github.com/euskadi31/wire"
	postv1 "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/app/search/service/internal/conf"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/meilisearch/meilisearch-go"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewSearchRepo, NewPostSourceRepo, NewPostClient)

// Data .
type Data struct {
	meili      meilisearch.ServiceManager
	postClient postv1.PostClient
	indexName  string
}

func NewPostClient(reg *etcd.Registry, c *conf.Data) postv1.PostClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.PostEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
	)
	if err != nil {
		panic(err)
	}
	return postv1.NewPostClient(conn)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, pc postv1.PostClient) (*Data, func(), error) {
	helper := log.NewHelper(logger)
	if c == nil {
		return nil, nil, errors.New("缺少 data 配置")
	}
	if c.Meilisearch == nil {
		return nil, nil, errors.New("缺少 data.meilisearch 配置")
	}
	ms := meilisearch.New(c.Meilisearch.Endpoint, meilisearch.WithAPIKey(c.Meilisearch.ApiKey))

	if c.Meilisearch.IndexName == "" {
		c.Meilisearch.IndexName = "posts"
	}

	d := &Data{
		meili:      ms,
		postClient: pc,
		indexName:  c.Meilisearch.IndexName,
	}

	if err := initIndexSettings(d); err != nil {
		helper.Warnf("初始化索引设置失败: %v", err)
	}

	cleanup := func() {
		helper.Info("closing search data resources")
	}
	return d, cleanup, nil
}
