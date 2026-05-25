package health

import (
	"context"
	"errors"

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
	endpoints := c.client.Endpoints()
	if len(endpoints) == 0 {
		return errors.New("etcd endpoints empty")
	}
	_, err := c.client.Maintenance.Status(ctx, endpoints[0])
	return err
}
