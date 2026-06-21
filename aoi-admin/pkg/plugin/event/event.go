package event

import (
	"context"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	"github.com/rei0721/go-scaffold/pkg/plugin/registry"
)

type Bus interface {
	Publish(context.Context, protocol.PushEventRequest) (protocol.PushEventResponse, error)
	Subscribe(context.Context, protocol.SubscribeEventRequest) (protocol.SubscribeEventResponse, error)
}

type Pusher interface {
	PushEvent(context.Context, protocol.PushEventRequest) (protocol.PushEventResponse, error)
}

type DirectBus struct {
	registry registry.Registry
	pusher   Pusher
}

func NewDirectBus(registry registry.Registry, pusher Pusher) *DirectBus {
	return &DirectBus{registry: registry, pusher: pusher}
}

func (b *DirectBus) Publish(ctx context.Context, req protocol.PushEventRequest) (protocol.PushEventResponse, error) {
	if b == nil || b.pusher == nil {
		return protocol.PushEventResponse{Accepted: true, Event: req.Event}, nil
	}
	return b.pusher.PushEvent(ctx, req)
}

func (b *DirectBus) Subscribe(ctx context.Context, req protocol.SubscribeEventRequest) (protocol.SubscribeEventResponse, error) {
	if b == nil || b.registry == nil {
		return protocol.SubscribeEventResponse{}, registry.ErrInvalid
	}
	accepted := make([]string, 0, len(req.Events))
	for _, eventName := range req.Events {
		sub, err := b.registry.SubscribeEvent(ctx, registry.Subscription{
			PluginID:   req.PluginID,
			InstanceID: req.InstanceID,
			Event:      eventName,
			Filters:    req.Filters,
		})
		if err != nil {
			return protocol.SubscribeEventResponse{}, err
		}
		accepted = append(accepted, sub.Event)
	}
	return protocol.SubscribeEventResponse{
		PluginID:   req.PluginID,
		InstanceID: req.InstanceID,
		Events:     accepted,
		Accepted:   true,
	}, nil
}
