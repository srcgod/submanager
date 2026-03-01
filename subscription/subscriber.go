package subscription

import (
	"github.com/nats-io/nats.go"
	natsgo "github.com/srcgod/newcommon/nats"
)

type Subscriber struct {
	client *natsgo.NatsClient
}

func NewSubscriber(client *natsgo.NatsClient) *Subscriber {
	return &Subscriber{client: client}
}

// Subscribe wraps underlying NATS client subscription.
func (s *Subscriber) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return s.client.Subscribe(subject, handler)
}
