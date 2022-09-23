# Redis Setup

## Kubernetes Setup

### Run the following script

```bash
./redis-k8s.sh
```

Add the following key in .nu.json

```json
  "redis": {
    "addr" :"redis-0.redis-service.redis.svc.cluster.local:6379"
  }
```

### Debug using redis-cli

```bash
kubectl exec -it redis-0 -n redis -- redis-cli
```

### Cleanup

```bash
kubectl delete ns redis
```

## Local Setup

### Run the following script

```bash
./redis-local.sh
```

Add the following key in .nu.json

```json
  "redis": {
    "addr" :"${YOUR_PRIVATE_IP}:6379"
  }
```

### Debug using redis-cli

```bash
redis-cli -h localhost -p 6379
# or
# docker exec -it redis  redis-cli
```

### Cleanup

```bash
docker kill redis
```