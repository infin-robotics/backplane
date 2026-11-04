package backplane

import (
	"context"
	"reflect"
)

// Option configures one invocation of [Backplane.Run]. Options are recognised
// separately from injectable resources and can only be defined by this package.
// If an option is repeated, the last value wins.
type Option interface {
	apply(*runOptions)
}

type runOptions struct {
	onMessage messageHandler
}

type messageHandler func(context.Context, Node, reflect.Type, any)

func (handler messageHandler) apply(options *runOptions) {
	options.onMessage = handler
}

// OnMessage observes each value accepted from a publisher, once before fan-out
// to subscribers or Latest. sender is the module's node in [Backplane.Graph];
// topic is the declared topic type, even when value has a different dynamic type.
// The context is the shared run context, not a per-message trace context.
//
// Calls are synchronous and serial within a topic, but may overlap across
// topics. The handler must be concurrency-safe, return promptly and not mutate
// message data. It adds backpressure and can delay shutdown; Run waits for it.
// Panics are not recovered. Observation does not acknowledge subscriber work,
// and a publisher's send may return before its callback runs.
//
// Values drained and dropped during cancellation are still observed, with a
// cancelled context. No callbacks run after Run returns. Passing nil disables
// observation, including any handler configured earlier in the same Run call.
func OnMessage(handler func(ctx context.Context, sender Node, topic reflect.Type, value any)) Option {
	return messageHandler(handler)
}
