package data

import (
	"errors"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/search-service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/meilisearch/meilisearch-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewSearchRepo, NewPostSourceRepo)

// Data .
type Data struct {
	meili      meilisearch.ServiceManager
	postClient postv1.PostClient
	indexName  string
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)
	if c == nil {
		return nil, nil, errors.New("缺少 data 配置")
	}
	if c.Meilisearch == nil {
		return nil, nil, errors.New("缺少 data.meilisearch 配置")
	}
	ms := meilisearch.New(c.Meilisearch.Endpoint, meilisearch.WithAPIKey(c.Meilisearch.ApiKey))

	postConn, err := grpc.NewClient(c.Clients.PostEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	if c.Meilisearch.IndexName == "" {
		c.Meilisearch.IndexName = "posts"
	}

	d := &Data{
		meili:      ms,
		postClient: postv1.NewPostClient(postConn),
		indexName:  c.Meilisearch.IndexName,
	}

	if err := initIndexSettings(d); err != nil {
		helper.Warnf("初始化索引设置失败: %v", err)
	}

	cleanup := func() {
		helper.Info("closing search data resources")
		if err := postConn.Close(); err != nil {
			helper.Errorf("关闭 post grpc 连接失败: %v", err)
		}
	}
	return d, cleanup, nil
}
