package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Register registers the service with etcd
func (sr *ServiceRegistry) Register(ctx context.Context) error {
	// Create lease
	lease, err := sr.client.Grant(ctx, 10) // 10 second TTL
	if err != nil {
		return err
	}
	sr.leaseID = lease.ID

	value, err := json.Marshal(sr.service)
	if err != nil {
		return err
	}

	// Put service info with lease
	_, err = sr.client.Put(ctx, sr.serviceKey, string(value), clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}

	// Keep lease alive
	keepAliveCh, err := sr.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return err
	}

	go sr.keepAlive(keepAliveCh)

	return nil
}

// keepAlive maintains the lease
func (sr *ServiceRegistry) keepAlive(keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		select {
		case <-sr.stopCh:
			return
		case resp := <-keepAliveCh:
			if resp == nil {
				// Lease expired or error occurred
				go sr.tryReregister()
				return
			}
		}
	}
}

// tryReregister attempts to register the service again
func (sr *ServiceRegistry) tryReregister() {
	ctx := context.Background()
	for {
		select {
		case <-sr.stopCh:
			return
		default:
			if err := sr.Register(ctx); err == nil {
				return
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// Discover looks up services by name
func (sr *ServiceRegistry) Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error) {
	prefix := fmt.Sprintf("/services/%s/", serviceName)
	resp, err := sr.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	services := make([]ServiceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var service ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			continue
		}
		services = append(services, service)
	}

	return services, nil
}

// Watch watches for service changes
func (sr *ServiceRegistry) Watch(serviceName string) clientv3.WatchChan {
	prefix := fmt.Sprintf("/services/%s/", serviceName)
	return sr.client.Watch(context.Background(), prefix, clientv3.WithPrefix())
}
