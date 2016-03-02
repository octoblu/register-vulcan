package vctl

import (
	"github.com/octoblu/vulcand-bundle/registry"
	"github.com/vulcand/vulcand/api"
	"github.com/vulcand/vulcand/engine"
)

// Vctl executes vctl commands
type Vctl interface {
	ServerUpsert(id, backendID, uri string) error
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
func (command *vctlClient) ServerUpsert(id, backendID, uri string) error {
	backendKey := engine.BackendKey{Id: backendID}

	oldServer, err := command.client.GetServer(engine.ServerKey{Id: id, BackendKey: backendKey})
	if err != nil {
		return command.doServerUpsert(id, uri, backendKey)
	}

	if uri != oldServer.URL {
		return command.doServerUpsert(id, uri, backendKey)
	}

	return nil
}

func (command *vctlClient) doServerUpsert(id, uri string, backendKey engine.BackendKey) error {
	server, err := engine.NewServer(id, uri)
	if err != nil {
		return err
	}

	return command.client.UpsertServer(backendKey, *server, 0)
}

func (command *vctlClient) ServerRm(id, backendID string) error {
	backendKey := engine.BackendKey{Id: backendID}
	serverKey := engine.ServerKey{BackendKey: backendKey, Id: id}

	return command.client.DeleteServer(serverKey)
}
