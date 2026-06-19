// Command consul-server-runbook detects a Consul server cluster's rolling-replacement
// plan and renders it through a template you supply (--template). See README.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

// Build information, injected at release time by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Step is one server replacement, exposed to the template.
type Step struct {
	I, N     int    // step number and total
	Role     string // "follower" or "LEADER"
	IsLeader bool
	Node     string // the Consul node name being replaced
	Host     string // its address (where the org provisions the new server)
	Prune    string // consul force-leave -prune <node>
	Dump     string // consul-state-dump ... > runs/after-step-N.json
	Diff     string // consul-state-diff runs/baseline.json runs/after-step-N.json
}

// Plan is the whole detected plan, exposed to the template.
type Plan struct {
	Leader       string // detected leader node name
	Nodes        string // space-joined HTTP addresses
	RunsDir      string
	ServerTail   string // consul-server-tail --nodes ... --expect N
	BaselineDump string // consul-state-dump ... > runs/baseline.json
	Steps        []Step
}

type server struct {
	Node   string
	Addr   string
	Leader bool
}

// raftServers returns the raft servers from the first reachable node.
func raftServers(nodes []string, timeout time.Duration) ([]server, error) {
	client := &http.Client{Timeout: timeout}
	var last error
	for _, addr := range nodes {
		resp, err := client.Get("http://" + addr + "/v1/operator/raft/configuration")
		if err != nil {
			last = err
			continue
		}
		var cfg struct {
			Servers []struct {
				Node    string `json:"Node"`
				Address string `json:"Address"`
				Leader  bool   `json:"Leader"`
			} `json:"Servers"`
		}
		err = json.NewDecoder(resp.Body).Decode(&cfg)
		resp.Body.Close()
		if err != nil {
			last = err
			continue
		}
		out := make([]server, 0, len(cfg.Servers))
		for _, s := range cfg.Servers {
			out = append(out, server{s.Node, s.Address, s.Leader})
		}
		return out, nil
	}
	return nil, fmt.Errorf("no node reachable (last error: %v)", last)
}

// buildPlan orders the servers followers-first / leader-last and fills the
// per-step Consul commands.
func buildPlan(servers []server, nodes []string, runsDir string) Plan {
	var followers, leaders []server
	for _, s := range servers {
		if s.Leader {
			leaders = append(leaders, s)
		} else {
			followers = append(followers, s)
		}
	}
	sort.Slice(followers, func(i, j int) bool { return followers[i].Node < followers[j].Node })
	order := append(followers, leaders...)

	nodesArg := strings.Join(nodes, " ")
	n := len(order)
	plan := Plan{
		Nodes:        nodesArg,
		RunsDir:      runsDir,
		ServerTail:   fmt.Sprintf("consul-server-tail --nodes %s --expect %d", nodesArg, n),
		BaselineDump: fmt.Sprintf("consul-state-dump %s > %s/baseline.json", nodesArg, runsDir),
	}
	if len(leaders) > 0 {
		plan.Leader = leaders[0].Node
	}
	for i, s := range order {
		host := s.Addr
		if idx := strings.LastIndex(host, ":"); idx >= 0 {
			host = host[:idx]
		}
		role := "follower"
		if s.Leader {
			role = "LEADER"
		}
		snap := fmt.Sprintf("%s/after-step-%d.json", runsDir, i+1)
		plan.Steps = append(plan.Steps, Step{
			I: i + 1, N: n, Role: role, IsLeader: s.Leader, Node: s.Node, Host: host,
			Prune: fmt.Sprintf("consul force-leave -prune %s", s.Node),
			Dump:  fmt.Sprintf("consul-state-dump %s > %s", nodesArg, snap),
			Diff:  fmt.Sprintf("consul-state-diff %s/baseline.json %s", runsDir, snap),
		})
	}
	return plan
}

func main() {
	tmplPath := flag.String("template", "", "path to the runbook template (required; see examples/runbook.md.tmpl)")
	runsDir := flag.String("runs-dir", "runs", "directory the generated commands write snapshots into")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request HTTP timeout")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: consul-server-runbook --template <file> [--runs-dir d] <host:port> [<host:port> ...]")
		fmt.Fprintln(os.Stderr, "  renders the detected leader-last plan through your template; see examples/runbook.md.tmpl")
	}
	flag.Parse()
	if *showVersion {
		fmt.Printf("consul-server-runbook %s (commit %s, built %s)\n", version, commit, date)
		return
	}
	if *tmplPath == "" {
		fmt.Fprintln(os.Stderr, "consul-server-runbook: --template is required (see examples/runbook.md.tmpl)")
		os.Exit(64)
	}
	nodes := flag.Args()
	if len(nodes) == 0 {
		flag.Usage()
		os.Exit(64)
	}

	data, err := os.ReadFile(*tmplPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "consul-server-runbook: %v\n", err)
		os.Exit(64)
	}
	tmpl, err := template.New("runbook").Parse(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "consul-server-runbook: %s: %v\n", *tmplPath, err)
		os.Exit(64)
	}

	servers, err := raftServers(nodes, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "consul-server-runbook: %v\n", err)
		os.Exit(1)
	}
	plan := buildPlan(servers, nodes, *runsDir)
	if plan.Leader == "" {
		fmt.Fprintln(os.Stderr, "consul-server-runbook: warning: no leader detected (election in progress?); ordered without a leader-last step")
	}
	if err := tmpl.Execute(os.Stdout, plan); err != nil {
		fmt.Fprintf(os.Stderr, "consul-server-runbook: %v\n", err)
		os.Exit(1)
	}
}
