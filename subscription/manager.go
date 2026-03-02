package subscription

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type SubscriptionManager[K comparable] struct {
	mu         sync.RWMutex
	subs       map[K]map[string]WSclient
	clientSubs map[string]map[K]struct{}
	natsSubs   map[K]*nats.Subscription
	sub        *Subscriber
	topicFunc  func(K) string
	handler    nats.MsgHandler
	logger     *logrus.Logger // TODO: remove logger
}

func NewSubscriptionManager[K comparable](topicFinc func(K) string, handler nats.MsgHandler, sub *Subscriber) *SubscriptionManager[K] {
	return &SubscriptionManager[K]{
		subs:       map[K]map[string]WSclient{},
		clientSubs: make(map[string]map[K]struct{}),
		natsSubs:   make(map[K]*nats.Subscription),
		sub:        sub,
		topicFunc:  topicFinc,
		handler:    handler,
		logger:     logrus.New(),
	}
}

func (s *SubscriptionManager[K]) AddClient(key K, sck WSclient) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.subs[key]) == 0 {
		topic := s.topicFunc(key)
		natsSub, err := s.sub.Subscribe(topic, s.handler)
		if err != nil {
			s.logger.WithField("key", key).Error("Nats subscribe error")
			return false, fmt.Errorf("failed to subscribe to NATS: %w", err)
		}

		s.logger.Debug("Created nats subscribe")

		s.natsSubs[key] = natsSub

		s.logger.WithField("key", key).Debug("Subscribed to NATS")
	}

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
		return nil, fmt.Errorf("Subscriptions is empty: %w", key)
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
				if sub, ok := m.natsSubs[key]; ok {
					if err := sub.Unsubscribe(); err != nil {
						m.logger.WithError(err).WithField("key", key).Error("Failed to unsubscribe from NATS")
					} else {
						delete(m.natsSubs, key)
						m.logger.WithField("key", key).Debug("Unsubscribed from NATS")
					}
				}
			}
		}
	}
	delete(m.clientSubs, clientID)
}
