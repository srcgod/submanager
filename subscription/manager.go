package subscription

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type SubscriptionManager[K comparable] struct {
	mu         sync.RWMutex
	subs       map[K]map[string]WSclient
	clientSubs map[string]map[K]struct{}
	logger     *logrus.Logger // TODO: remove logger
}

func NewSubscriptionManager[K comparable]() *SubscriptionManager[K] {
	return &SubscriptionManager[K]{
		subs:       map[K]map[string]WSclient{},
		clientSubs: make(map[string]map[K]struct{}),
		logger:     logrus.New(),
	}
}

func (s *SubscriptionManager[K]) AddClient(key K, sck WSclient) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientID := sck.ID()
	if _, ok := s.subs[key]; !ok {
		s.subs[key] = make(map[string]WSclient)
	}
	if _, ok := s.clientSubs[clientID]; !ok {
		s.clientSubs[clientID] = make(map[K]struct{})
	}

	if _, ok := s.subs[key][clientID]; ok {
		s.logger.Warn("the socket is already registered to : ", key)
		return true, nil
	}

	s.subs[key][clientID] = sck
	s.clientSubs[clientID][key] = struct{}{}

	return true, nil
}

func (m *SubscriptionManager[K]) ClientSubs(key K) (map[string]WSclient, error) {
	if _, ok := m.subs[key]; !ok {
		return nil, fmt.Errorf("Subscriptions is empty")
	}
	return m.subs[key], nil
}

func (m *SubscriptionManager[K]) RemoveClient(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys, ok := m.clientSubs[clientID]
	if !ok {
		return
	}
	for key := range keys {
		if clients, ok := m.subs[key]; ok {
			delete(clients, clientID)
			if len(clients) == 0 {
				delete(m.subs, key)
			}
		}
	}
	delete(m.clientSubs, clientID)
}
