package serve

import "github.com/gnoswap-labs/grc20-register/events"

// Events is the interface for event passing
type Events interface {
	// Subscribe subscribes to specific events
	Subscribe([]events.Type) *events.Subscription

	// CancelSubscription cancels the given subscription
	CancelSubscription(events.SubscriptionID)
}
