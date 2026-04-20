package data

import (
	"context"
	"errors"

	commentv1 "github.com/satiu123/GoPalette/api/comment/v1"
	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/conf"

	"github.com/euskadi31/wire"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	kratosmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUserClient, NewPostClient, NewCommentClient)

// Data .
type Data struct {
	userClient    userv1.UserClient
	postClient    postv1.PostClient
	commentClient commentv1.CommentClient
}

func NewUserClient(reg *etcd.Registry, c *conf.Data) userv1.UserClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.UserEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(kratosmd.Client(), tracing.Client()),
	)
	if err != nil {
		panic(err)
	}
	return userv1.NewUserClient(conn)
}

func NewPostClient(reg *etcd.Registry, c *conf.Data) postv1.PostClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.PostEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(kratosmd.Client(), tracing.Client()),
	)
	if err != nil {
		panic(err)
	}
	return postv1.NewPostClient(conn)
}

func NewCommentClient(reg *etcd.Registry, c *conf.Data) commentv1.CommentClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.CommentEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(kratosmd.Client(), tracing.Client()),
	)
	if err != nil {
		panic(err)
	}
	return commentv1.NewCommentClient(conn)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, uc userv1.UserClient, pc postv1.PostClient, cc commentv1.CommentClient) (*Data, func(), error) {
	helper := log.NewHelper(logger)
	if c == nil {
		return nil, nil, errors.New("缺少 data 配置")
	}
	if c.Clients == nil {
		return nil, nil, errors.New("缺少 data.clients 配置")
	}
	d := &Data{
		userClient:    uc,
		postClient:    pc,
		commentClient: cc,
	}
	return d, func() {
		helper.Info("closing bff data resources")
	}, nil
}

func (d *Data) UserClient() userv1.UserClient {
	return d.userClient
}

func (d *Data) PostClient() postv1.PostClient {
	return d.postClient
}

func (d *Data) CommentClient() commentv1.CommentClient {
	return d.commentClient
}
