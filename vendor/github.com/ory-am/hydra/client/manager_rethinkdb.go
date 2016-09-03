package client

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-errors/errors"
	"github.com/ory-am/fosite"
	"github.com/ory-am/fosite/hash"
	"github.com/ory-am/hydra/pkg"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	r "gopkg.in/dancannon/gorethink.v2"
)

type RethinkManager struct {
	Session *r.Session
	Table   r.Term
	sync.RWMutex

	Clients map[string]Client
	Hasher  hash.Hasher
}

func (m *RethinkManager) GetConcreteClient(id string) (*Client, error) {
	m.RLock()
	defer m.RUnlock()

	c, ok := m.Clients[id]
	if !ok {
		return nil, errors.New(pkg.ErrNotFound)
	}
	return &c, nil
}

func (m *RethinkManager) GetClient(id string) (fosite.Client, error) {
	return m.GetConcreteClient(id)
}

func (m *RethinkManager) Authenticate(id string, secret []byte) (*Client, error) {
	m.RLock()
	defer m.RUnlock()

	c, ok := m.Clients[id]
	if !ok {
		return nil, errors.New(pkg.ErrNotFound)
	}

	if err := m.Hasher.Compare(c.GetHashedSecret(), secret); err != nil {
		return nil, errors.New(err)
	}

	return &c, nil
}

func (m *RethinkManager) CreateClient(c *Client) error {
	if c.ID == "" {
		c.ID = uuid.New()
	}

	hash, err := m.Hasher.Hash([]byte(c.Secret))
	if err != nil {
		return errors.New(err)
	}
	c.Secret = string(hash)

	if err := m.publishCreate(c); err != nil {
		return err
	}

	return nil
}

func (m *RethinkManager) DeleteClient(id string) error {
	if err := m.publishDelete(id); err != nil {
		return err
	}

	return nil
}

func (m *RethinkManager) GetClients() (clients map[string]Client, err error) {
	m.RLock()
	defer m.RUnlock()
	clients = make(map[string]Client)
	for _, c := range m.Clients {
		clients[c.ID] = c
	}

	return clients, nil
}

func (m *RethinkManager) ColdStart() error {
	m.Clients = map[string]Client{}
	clients, err := m.Table.Run(m.Session)
	if err != nil {
		return errors.New(err)
	}

	var client Client
	m.Lock()
	defer m.Unlock()
	for clients.Next(&client) {
		m.Clients[client.ID] = client
	}

	return nil
}

func (m *RethinkManager) publishCreate(client *Client) error {
	if _, err := m.Table.Insert(client).RunWrite(m.Session); err != nil {
		return errors.New(err)
	}
	return nil
}

func (m *RethinkManager) publishDelete(id string) error {
	if _, err := m.Table.Get(id).Delete().RunWrite(m.Session); err != nil {
		return errors.New(err)
	}
	return nil
}

func (m *RethinkManager) Watch(ctx context.Context) {
	go pkg.Retry(time.Second*15, time.Minute, func() error {
		clients, err := m.Table.Changes().Run(m.Session)
		if err != nil {
			return errors.New(err)
		}
		defer clients.Close()

		var update map[string]*Client
		for clients.Next(&update) {
			logrus.Debug("Received update from RethinkDB Cluster in OAuth2 client manager.")
			newVal := update["new_val"]
			oldVal := update["old_val"]
			m.Lock()
			if newVal == nil && oldVal != nil {
				delete(m.Clients, oldVal.GetID())
			} else if newVal != nil && oldVal != nil {
				delete(m.Clients, oldVal.GetID())
				m.Clients[newVal.GetID()] = *newVal
			} else {
				m.Clients[newVal.GetID()] = *newVal
			}
			m.Unlock()
		}

		if clients.Err() != nil {
			err = errors.New(clients.Err())
			pkg.LogError(err)
			return err
		}
		return nil
	})
}
