# consul-server-runbook

consul-server-runbook is not a turnkey runbook generator. It detects a live Consul
server cluster's safe rolling-replacement plan (which servers, in followers-first
/ leader-last order) and the verification to run at each step
([consul-state-dump / consul-state-diff](https://github.com/zinrai/consul-state-verify),
the [consul-server-tail](https://github.com/zinrai/consul-server-tail)
checkpoints), and renders that plan through a template you supply.

The server-replacement itself (stop the old server, provision and start the new
one at the same address) is your business domain. The tool cannot know it, so it
lives in your template. The boundary:

- **The tool generates** (Consul-side, derivable from the cluster): the leader,
  the followers-first / leader-last order, `consul force-leave -prune <node>`, the
  per-step `consul-state-dump` / `consul-state-diff` commands, and the
  consul-server-tail checkpoints.
- **You generate** (OS / infrastructure, organization-specific): stopping,
  provisioning, setting up, and starting the server. You write it once, in your
  template.

Read-only: it reads `/v1/operator/raft/configuration` and nothing else.

## Usage

```
consul-server-runbook --template runbook.md.tmpl \
  10.0.0.1:8500 10.0.0.2:8500 10.0.0.3:8500 10.0.0.4:8500 10.0.0.5:8500 > runbook.md
```

`--template` is required: there is no built-in default, because the runbook's
content is yours. Start from [examples/runbook.md.tmpl](examples/runbook.md.tmpl):
copy it and write your server-replacement steps into the marked slot.

```
--template   path to the runbook template (required)
--runs-dir   directory the generated commands write snapshots into (default: runs)
--timeout    per-request HTTP timeout in seconds (default 5s)
```

Node addresses are positional. Run it after the cluster is healthy (the detected
leader is the one to replace last; replacing followers does not move leadership).

Values that are not detected from the cluster (the target version, a change
ticket, a maintenance window) are not tool inputs; set them in your template
(e.g. `{{$target := "1.22"}}` once near the top, referenced as `{{$target}}`).

## Template data

The template is [Go `text/template`](https://pkg.go.dev/text/template), rendered
against the detected plan:

```
.Leader        detected leader node name
.Nodes         the node addresses, space-joined
.RunsDir       the snapshot directory
.ServerTail    consul-server-tail --nodes ... --expect N
.BaselineDump  consul-state-dump ... > <runs>/baseline.json
.Steps         one per server, followers first then the leader:
    .I .N        step number and total
    .Role        "follower" or "LEADER"
    .IsLeader    bool (use for the transfer-leadership note)
    .Node        node name being replaced
    .Host        its address (where you provision the new server)
    .Prune       consul force-leave -prune <node>
    .Dump        consul-state-dump ... > <runs>/after-step-N.json
    .Diff        consul-state-diff <runs>/baseline.json <runs>/after-step-N.json
```

## License

This project is licensed under the [MIT License](./LICENSE).
