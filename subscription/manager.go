package subscription

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type SubscriptionManager[K comparable] struct {
	mu         sync.RWMutex
	subs       map[K]map[string]WSclient // ключ -> клиенты
	clientSubs map[string]map[K]struct{} // clientID -> множество ключей

	natsSubs   map[K][]*nats.Subscription // ключ -> список подписок NATS
	sub        *Subscriber
	topicsFunc *func(K) []string
	handler    *nats.MsgHandler
	logger     *logrus.Logger
}

func NewSubscriptionManager[K comparable](
	topicsFunc *func(K) []string,
	handler *nats.MsgHandler,
	sub *Subscriber,
	logger *logrus.Logger,
) *SubscriptionManager[K] {
	m := &SubscriptionManager[K]{
		subs:       make(map[K]map[string]WSclient),
		clientSubs: make(map[string]map[K]struct{}),
		logger:     logger,
	}
	if sub != nil && topicsFunc != nil {
		m.sub = sub
		m.topicsFunc = topicsFunc
		m.handler = handler
		m.natsSubs = make(map[K][]*nats.Subscription)
	}
	return m
}

func (m *SubscriptionManager[K]) AddClient(key K, client WSclient) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	clientID := client.ID()

	if _, ok := m.subs[key]; !ok {
		m.subs[key] = make(map[string]WSclient)
	}
	if _, ok := m.clientSubs[clientID]; !ok {
		m.clientSubs[clientID] = make(map[K]struct{})
	}

	if _, ok := m.subs[key][clientID]; ok {
		m.logger.WithField("key", key).Warn("client already subscribed to this key")
		return true, nil
	}

	if m.natsSubs != nil && len(m.subs[key]) == 0 {
		topics := (*m.topicsFunc)(key)
		subs := make([]*nats.Subscription, 0, len(topics))

		for _, topic := range topics {
			natsSub, err := m.sub.Subscribe(topic, *m.handler)
			if err != nil {
				for _, s := range subs {
					_ = s.Unsubscribe()
				}
				m.logger.WithError(err).WithField("key", key).Error("failed to subscribe to NATS")
				return false, fmt.Errorf("failed to subscribe to NATS: %w", err)
			}
			subs = append(subs, natsSub)
			m.logger.WithField("topic", topic).Debug("subscribed to NATS")
		}
		m.natsSubs[key] = subs
		m.logger.WithField("key", key).Debugf("subscribed to %d NATS topics", len(subs))
	}

	// Добавляем клиента
	m.subs[key][clientID] = client
	m.clientSubs[clientID][key] = struct{}{}

	return true, nil
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
				// Если есть NATS-подписки для этого ключа — отписываемся
				if m.natsSubs != nil {
					if subs, ok := m.natsSubs[key]; ok {
						for _, sub := range subs {
							_ = sub.Unsubscribe()
						}
						delete(m.natsSubs, key)
						m.logger.WithField("key", key).Debug("unsubscribed from NATS")
					}
				}
			}
		}
	}

	delete(m.clientSubs, clientID)
}

func (m *SubscriptionManager[K]) ClientSubs(key K) map[string]WSclient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients, ok := m.subs[key]
	if !ok {
		return nil
	}
	result := make(map[string]WSclient, len(clients))
	for id, cl := range clients {
		result[id] = cl
	}
	return result
}
