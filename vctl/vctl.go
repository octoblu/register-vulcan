package vctl

import (
	"time"

	"github.com/octoblu/vulcand-bundle/registry"
	"github.com/vulcand/vulcand/api"
	"github.com/vulcand/vulcand/engine"
)

// Vctl executes vctl commands
type Vctl interface {
	ServerUpsert(id, backendID, uri string, ttl time.Duration) error
	ServerRm(id, backendID string) error
}

type vctlClient struct {
	client *api.Client
}

// New returns a new Vctl instance
func New(vulcanURL string) (Vctl, error) {
	reg, err := registry.GetRegistry()
	if err != nil {
		return nil, err
	}

	client := api.NewClient(vulcanURL, reg)
	return &vctlClient{client}, nil
}

// ServerUpsert upserts a new server
func (command *vctlClient) ServerUpsert(id, backendID, uri string, ttl time.Duration) error {
	server, err := engine.NewServer(id, uri)
	if err != nil {
		return err
	}

	backendKey := engine.BackendKey{Id: backendID}

	return command.client.UpsertServer(backendKey, *server, ttl)
}

func (command *vctlClient) ServerRm(id, backendID string) error {
	backendKey := engine.BackendKey{Id: backendID}
	serverKey := engine.ServerKey{BackendKey: backendKey, Id: id}

	return command.client.DeleteServer(serverKey)
}
