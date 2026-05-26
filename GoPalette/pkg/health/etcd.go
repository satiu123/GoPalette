package health

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdChecker struct {
	client *clientv3.Client
}

func NewEtcdChecker(client *clientv3.Client) *EtcdChecker {
	return &EtcdChecker{
		client: client,
	}
}

func (c *EtcdChecker) Name() string {
	return "etcd"
}

func (c *EtcdChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err := c.client.Get(ctx, "health", clientv3.WithLimit(1))
	return err
}
