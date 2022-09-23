# Redis Cluster Setup on Kubernetes

## Run the following command to setup Redis Cluster

### Deploy k8 resources

```bash
kubectl apply -f ../redis-cluster/
```

### Create cluster using redis-cli

```bash
 kubectl exec -it redis-cluster-0 -n redis  -- redis-cli --cluster create --cluster-replicas 1 $(kubectl get pods -n redis -o jsonpath='{range.items[*]}{.status.podIP}:6379 {end}')
```

### Add the cluster addresses in config file

```json
  "redis": {
    "addr" :"redis-cluster-0.redis-cluster.redis.svc.cluster.local:6379,redis-cluster-1.redis-cluster.redis.svc.cluster.local:6379,redis-cluster-2.redis-cluster.redis.svc.cluster.local:6379,redis-cluster-3.redis-cluster.redis.svc.cluster.local:6379,redis-cluster-4.redis-cluster.redis.svc.cluster.local:6379,redis-cluster-5.redis-cluster.redis.svc.cluster.local:6379"
  }
```
