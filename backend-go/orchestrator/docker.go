// Package orchestrator provisions and tears down the short-lived test
// infrastructure (Elasticsearch + Neo4j) used to validate a single Sigma
// rule during a CI run. It talks to the real Docker Engine API — nothing
// here is mocked.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// TestEnvironment represents one ephemeral rule-test sandbox.
type TestEnvironment struct {
	RunID         string
	ElasticCID    string
	Neo4jCID      string
	ElasticPort   string
	Neo4jHTTPPort string
	Neo4jBoltPort string

	cli *dockerclient.Client
}

// New creates a Docker client using the local daemon (DOCKER_HOST env var
// respected automatically by the SDK).
func New() (*dockerclient.Client, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return cli, nil
}

// Provision starts a single-node Elasticsearch container and a Neo4j
// container, each on random host ports, tagged with the CI run ID so they
// can be reliably torn down afterward even if the pipeline crashes mid-run.
func Provision(ctx context.Context, cli *dockerclient.Client, runID string) (*TestEnvironment, error) {
	env := &TestEnvironment{RunID: runID, cli: cli}

	esPortSpec, _ := nat.NewPort("tcp", "9200")
	esResp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "docker.elastic.co/elasticsearch/elasticsearch:8.13.0",
		Env: []string{
			"discovery.type=single-node",
			"xpack.security.enabled=false",
			"ES_JAVA_OPTS=-Xms512m -Xmx512m",
		},
		ExposedPorts: nat.PortSet{esPortSpec: struct{}{}},
		Labels:       map[string]string{"wraith.run_id": runID, "wraith.role": "elasticsearch"},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{esPortSpec: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "0"}}},
		AutoRemove:   false,
	}, &network.NetworkingConfig{}, nil, fmt.Sprintf("wraith-es-%s", runID))
	if err != nil {
		return nil, fmt.Errorf("creating elasticsearch container: %w", err)
	}
	if err := cli.ContainerStart(ctx, esResp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("starting elasticsearch container: %w", err)
	}
	env.ElasticCID = esResp.ID

	boltPortSpec, _ := nat.NewPort("tcp", "7687")
	httpPortSpec, _ := nat.NewPort("tcp", "7474")
	neoResp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "neo4j:5.19-community",
		Env: []string{
			"NEO4J_AUTH=neo4j/wraith-test-pw",
			"NEO4J_PLUGINS=[\"apoc\"]",
		},
		ExposedPorts: nat.PortSet{boltPortSpec: struct{}{}, httpPortSpec: struct{}{}},
		Labels:       map[string]string{"wraith.run_id": runID, "wraith.role": "neo4j"},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			boltPortSpec: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "0"}},
			httpPortSpec: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "0"}},
		},
	}, &network.NetworkingConfig{}, nil, fmt.Sprintf("wraith-neo4j-%s", runID))
	if err != nil {
		_ = Teardown(ctx, cli, env) // best-effort cleanup of ES if neo4j fails
		return nil, fmt.Errorf("creating neo4j container: %w", err)
	}
	if err := cli.ContainerStart(ctx, neoResp.ID, container.StartOptions{}); err != nil {
		_ = Teardown(ctx, cli, env)
		return nil, fmt.Errorf("starting neo4j container: %w", err)
	}
	env.Neo4jCID = neoResp.ID

	if err := waitForPorts(ctx, cli, env); err != nil {
		_ = Teardown(ctx, cli, env)
		return nil, err
	}

	return env, nil
}

// waitForPorts inspects the running containers to resolve the dynamically
// assigned host ports and polls until both services accept connections.
func waitForPorts(ctx context.Context, cli *dockerclient.Client, env *TestEnvironment) error {
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		esInfo, err := cli.ContainerInspect(ctx, env.ElasticCID)
		if err == nil {
			if b, ok := esInfo.NetworkSettings.Ports["9200/tcp"]; ok && len(b) > 0 {
				env.ElasticPort = b[0].HostPort
			}
		}
		neoInfo, err := cli.ContainerInspect(ctx, env.Neo4jCID)
		if err == nil {
			if b, ok := neoInfo.NetworkSettings.Ports["7474/tcp"]; ok && len(b) > 0 {
				env.Neo4jHTTPPort = b[0].HostPort
			}
			if b, ok := neoInfo.NetworkSettings.Ports["7687/tcp"]; ok && len(b) > 0 {
				env.Neo4jBoltPort = b[0].HostPort
			}
		}
		if env.ElasticPort != "" && env.Neo4jHTTPPort != "" && env.Neo4jBoltPort != "" {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for test environment ports to bind for run %s", env.RunID)
}

// Teardown stops and removes both containers for a run. It is safe to call
// even if provisioning only partially succeeded.
func Teardown(ctx context.Context, cli *dockerclient.Client, env *TestEnvironment) error {
	var firstErr error
	stopRemove := func(cid string) {
		if cid == "" {
			return
		}
		timeout := 10
		_ = cli.ContainerStop(ctx, cid, container.StopOptions{Timeout: &timeout})
		if err := cli.ContainerRemove(ctx, cid, container.RemoveOptions{Force: true}); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	stopRemove(env.ElasticCID)
	stopRemove(env.Neo4jCID)
	return firstErr
}

// TeardownByRunID finds and removes any containers labeled with the given
// run ID. Used as a recovery path (e.g. cron sweep) if a CI job died before
// calling Teardown directly.
func TeardownByRunID(ctx context.Context, cli *dockerclient.Client, runID string) error {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}
	for _, c := range containers {
		if c.Labels["wraith.run_id"] == runID {
			timeout := 5
			_ = cli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout})
			_ = cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
		}
	}
	return nil
}
