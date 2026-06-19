# Consul server rolling upgrade (fixed IP, fresh provision) (generated)

5 servers to hashicorp/consul:1.22.7. Detected leader: **consul-server-2**.
Replacement order (followers first, leader last): `consul-server-0` `consul-server-1` `consul-server-3` `consul-server-4` `consul-server-2` (LEADER) 

> Per-step `consul-state-diff` verdict: **PASS** continue / **INVESTIGATE** read the loss and record it / **FAIL** stop.

## 0. Before you start (once)

Bring up two monitors, seed a stable canary key for client-tail, and take the baseline:

```
export PATH="$HOME/bin:$PATH"
export CONSUL_SERVER_IMAGE=hashicorp/consul:1.22.7
mkdir -p runs

# a stable canary key for client-tail (a fixed key no churn touches)
docker compose exec -T consul-client-0 consul kv put canary/keepalive ok

# in two other terminals, keep both visible for the whole window:
#   server-side quorum (leader present, voters, index convergence)
consul-server-tail --nodes 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 --expect 5
#   the old clients' local access path (the residual the servers cannot see)
consul-client-tail --inventory clients.json

consul-state-dump 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 > runs/baseline.json
```

## Step 1 / 5: follower `consul-server-0` (172.28.0.10)

Shut the old server down and bring up a fresh hashicorp/consul:1.22.7 server at the same IP (new
disk, so a new node-id):

```
docker compose rm -sf consul-server-0
docker compose exec -T consul-client-0 consul force-leave -prune consul-server-0
docker volume rm hashicorp-consul-sandbox_consul-server-0-data
docker compose up -d consul-server-0
```

Wait on `consul-server-tail` for the cluster to re-stabilise (leader present,
voters=5, indexes converged) and check `consul-client-tail` is still green, then
verify the state:

```
consul-state-dump 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 > runs/after-step-1.json
consul-state-diff runs/baseline.json runs/after-step-1.json
```

## Step 2 / 5: follower `consul-server-1` (172.28.0.11)

Shut the old server down and bring up a fresh hashicorp/consul:1.22.7 server at the same IP (new
disk, so a new node-id):

```
docker compose rm -sf consul-server-1
docker compose exec -T consul-client-0 consul force-leave -prune consul-server-1
docker volume rm hashicorp-consul-sandbox_consul-server-1-data
docker compose up -d consul-server-1
```

Wait on `consul-server-tail` for the cluster to re-stabilise (leader present,
voters=5, indexes converged) and check `consul-client-tail` is still green, then
verify the state:

```
consul-state-dump 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 > runs/after-step-2.json
consul-state-diff runs/baseline.json runs/after-step-2.json
```

## Step 3 / 5: follower `consul-server-3` (172.28.0.13)

Shut the old server down and bring up a fresh hashicorp/consul:1.22.7 server at the same IP (new
disk, so a new node-id):

```
docker compose rm -sf consul-server-3
docker compose exec -T consul-client-0 consul force-leave -prune consul-server-3
docker volume rm hashicorp-consul-sandbox_consul-server-3-data
docker compose up -d consul-server-3
```

Wait on `consul-server-tail` for the cluster to re-stabilise (leader present,
voters=5, indexes converged) and check `consul-client-tail` is still green, then
verify the state:

```
consul-state-dump 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 > runs/after-step-3.json
consul-state-diff runs/baseline.json runs/after-step-3.json
```

## Step 4 / 5: follower `consul-server-4` (172.28.0.14)

Shut the old server down and bring up a fresh hashicorp/consul:1.22.7 server at the same IP (new
disk, so a new node-id):

```
docker compose rm -sf consul-server-4
docker compose exec -T consul-client-0 consul force-leave -prune consul-server-4
docker volume rm hashicorp-consul-sandbox_consul-server-4-data
docker compose up -d consul-server-4
```

Wait on `consul-server-tail` for the cluster to re-stabilise (leader present,
voters=5, indexes converged) and check `consul-client-tail` is still green, then
verify the state:

```
consul-state-dump 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 > runs/after-step-4.json
consul-state-diff runs/baseline.json runs/after-step-4.json
```

## Step 5 / 5: LEADER `consul-server-2` (172.28.0.12)

> This node is the leader; stopping it triggers an election. Confirm a new leader is in
> place before pruning.

Shut the old server down and bring up a fresh hashicorp/consul:1.22.7 server at the same IP (new
disk, so a new node-id):

```
docker compose rm -sf consul-server-2
docker compose exec -T consul-client-0 consul force-leave -prune consul-server-2
docker volume rm hashicorp-consul-sandbox_consul-server-2-data
docker compose up -d consul-server-2
```

Wait on `consul-server-tail` for the cluster to re-stabilise (leader present,
voters=5, indexes converged) and check `consul-client-tail` is still green, then
verify the state:

```
consul-state-dump 127.0.0.1:8500 127.0.0.1:8501 127.0.0.1:8502 127.0.0.1:8503 127.0.0.1:8504 > runs/after-step-5.json
consul-state-diff runs/baseline.json runs/after-step-5.json
```

## Done

All steps PASS, every server on hashicorp/consul:1.22.7 with a new node-id at the same IP, the
leader present, voters=5. Then upgrade the clients (servers first, clients
last).
