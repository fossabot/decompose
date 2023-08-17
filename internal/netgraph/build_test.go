package netgraph_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/s0rg/decompose/internal/netgraph"
	"github.com/s0rg/decompose/internal/node"
)

type testClient struct {
	Err  error
	Data []*netgraph.Container
}

func (tc *testClient) Containers(
	_ context.Context,
	_ netgraph.NetProto,
	fn func(int, int),
) ([]*netgraph.Container, error) {
	if tc.Err != nil {
		return nil, tc.Err
	}

	l := len(tc.Data)

	fn(0, l)

	if l > 1 {
		fn(l/2, l)
	}

	fn(l, l)

	return tc.Data, nil
}

type testBuilder struct {
	Err   error
	Nodes int
	Edges int
}

func (tb *testBuilder) AddNode(_ *node.Node) error {
	if tb.Err != nil {
		return tb.Err
	}

	tb.Nodes++

	return nil
}

func (tb *testBuilder) AddEdge(_, _ string, _ node.Port) {
	tb.Edges++
}

func TestBuildError(t *testing.T) {
	t.Parallel()

	myErr := errors.New("test error")
	cli := &testClient{Err: myErr}

	err := netgraph.Build(cli, nil, netgraph.ALL, "", false)
	if err == nil {
		t.Fatal("err is nil")
	}

	if !errors.Is(err, myErr) {
		t.Fatalf("unknown error, want: %v got: %v", myErr, err)
	}
}

func TestBuildOneConainer(t *testing.T) {
	t.Parallel()

	cli := &testClient{Data: []*netgraph.Container{
		{},
	}}

	bld := &testBuilder{}

	if err := netgraph.Build(cli, bld, netgraph.ALL, "", false); err != nil {
		t.Fatalf("err = %v", err)
	}

	if bld.Nodes > 0 {
		t.Fail()
	}
}

func makeContainer(name, ip string) *netgraph.Container {
	return &netgraph.Container{
		ID:    name + "-id",
		Name:  name,
		Image: name + "-image:latest",
		Endpoints: map[string]string{
			ip: "test-net",
		},
	}
}

func testClientWithEnv() netgraph.ContainerClient {
	node1 := net.ParseIP("1.1.1.1")
	node2 := net.ParseIP("1.1.1.2")
	node3 := net.ParseIP("1.1.1.3")
	external := net.ParseIP("2.2.2.1")

	cli := &testClient{Data: []*netgraph.Container{
		makeContainer("1", node1.String()),
		makeContainer("2", node2.String()),
		makeContainer("3", node3.String()),
	}}

	// node 1
	cli.Data[0].SetConnections([]*netgraph.Connection{
		{LocalPort: 1, Kind: netgraph.TCP},                                     // listen 1
		{RemoteIP: node2, LocalPort: 10, RemotePort: 2, Kind: netgraph.TCP},    // connected to node2:2
		{RemoteIP: external, LocalPort: 10, RemotePort: 1, Kind: netgraph.TCP}, // connected to external:1
	})

	// node 2
	cli.Data[1].SetConnections([]*netgraph.Connection{
		{LocalPort: 2, Kind: netgraph.TCP},                                     // listen 2
		{RemoteIP: node3, LocalPort: 10, RemotePort: 3, Kind: netgraph.TCP},    // connected to node3:3
		{RemoteIP: external, LocalPort: 10, RemotePort: 2, Kind: netgraph.TCP}, // connected to external:2
	})

	// node 3
	cli.Data[2].SetConnections([]*netgraph.Connection{
		{LocalPort: 3, Kind: netgraph.TCP},                                     // listen 3
		{RemoteIP: node1, LocalPort: 10, RemotePort: 1, Kind: netgraph.TCP},    // connected to node1:1
		{RemoteIP: external, LocalPort: 10, RemotePort: 3, Kind: netgraph.TCP}, // connected to external:3
	})

	return cli
}

func TestBuildSimple(t *testing.T) {
	t.Parallel()

	cli := testClientWithEnv()
	bld := &testBuilder{}

	if err := netgraph.Build(cli, bld, netgraph.ALL, "", false); err != nil {
		t.Fatalf("err = %v", err)
	}

	if bld.Nodes != 4 || bld.Edges != 6 {
		t.Fail()
	}
}

func TestBuildFollow(t *testing.T) {
	t.Parallel()

	cli := testClientWithEnv()
	bld := &testBuilder{}

	if err := netgraph.Build(cli, bld, netgraph.ALL, "1", false); err != nil {
		t.Fatalf("err = %v", err)
	}

	if bld.Nodes != 3 || bld.Edges != 3 {
		t.Fail()
	}
}

func TestBuildLocal(t *testing.T) {
	t.Parallel()

	cli := testClientWithEnv()
	bld := &testBuilder{}

	if err := netgraph.Build(cli, bld, netgraph.ALL, "", true); err != nil {
		t.Fatalf("err = %v", err)
	}

	if bld.Nodes != 3 || bld.Edges != 3 {
		t.Fail()
	}
}

func TestBuildNoNodes(t *testing.T) {
	t.Parallel()

	cli := testClientWithEnv()
	bld := &testBuilder{}

	if err := netgraph.Build(cli, bld, netgraph.ALL, "4", false); err != nil {
		t.Fatalf("err = %v", err)
	}

	if bld.Nodes > 0 || bld.Edges > 0 {
		t.Fail()
	}
}

func TestBuildNodeError(t *testing.T) {
	t.Parallel()

	myErr := errors.New("test error")
	cli := testClientWithEnv()
	bld := &testBuilder{Err: myErr}

	err := netgraph.Build(cli, bld, netgraph.ALL, "", false)
	if err == nil {
		t.Fatal("err is nil")
	}

	if !errors.Is(err, myErr) {
		t.Fatalf("unknown error, want: %v got: %v", myErr, err)
	}
}