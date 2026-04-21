package transport

import (
	"testing"

	"github.com/hashicorp/consul/api"
)

func TestHTTPTransportNextTargetHostRoundRobin(t *testing.T) {
	tp := &HTTPTransport{}
	services := []*api.ServiceEntry{
		{
			Service: &api.AgentService{
				Address: "10.0.0.1",
				Port:    35000,
			},
		},
		{
			Service: &api.AgentService{
				Address: "10.0.0.2",
				Port:    35001,
			},
		},
		{
			Service: &api.AgentService{
				Address: "10.0.0.3",
				Port:    35002,
			},
		},
	}

	got1, ok := tp.nextTargetHost(services)
	if !ok {
		t.Fatal("expected first target host")
	}
	if got1 != "10.0.0.1:35000" {
		t.Fatalf("unexpected first target host: %s", got1)
	}

	got2, ok := tp.nextTargetHost(services)
	if !ok {
		t.Fatal("expected second target host")
	}
	if got2 != "10.0.0.2:35001" {
		t.Fatalf("unexpected second target host: %s", got2)
	}

	got3, ok := tp.nextTargetHost(services)
	if !ok {
		t.Fatal("expected third target host")
	}
	if got3 != "10.0.0.3:35002" {
		t.Fatalf("unexpected third target host: %s", got3)
	}

	got4, ok := tp.nextTargetHost(services)
	if !ok {
		t.Fatal("expected fourth target host")
	}
	if got4 != "10.0.0.1:35000" {
		t.Fatalf("unexpected fourth target host: %s", got4)
	}
}

func TestHTTPTransportNextTargetHostSkipsInvalidEntries(t *testing.T) {
	tp := &HTTPTransport{}
	services := []*api.ServiceEntry{
		nil,
		{
			Service: &api.AgentService{
				Address: "",
				Port:    35000,
			},
		},
		{
			Node: &api.Node{
				Address: "10.0.0.9",
			},
			Service: &api.AgentService{
				Port: 35009,
			},
		},
	}

	got, ok := tp.nextTargetHost(services)
	if !ok {
		t.Fatal("expected target host from node fallback")
	}
	if got != "10.0.0.9:35009" {
		t.Fatalf("unexpected target host: %s", got)
	}
}

func TestHTTPTransportNextTargetHostReturnsFalseWhenNoUsableService(t *testing.T) {
	tp := &HTTPTransport{}

	got, ok := tp.nextTargetHost([]*api.ServiceEntry{
		{
			Service: &api.AgentService{
				Address: "",
				Port:    0,
			},
		},
	})
	if ok {
		t.Fatalf("expected no target host, got %s", got)
	}
}
