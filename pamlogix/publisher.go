package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

type PublisherEvent struct {
	Name      string            `json:"name,omitempty"`
	Id        string            `json:"id,omitempty"`
	Timestamp int64             `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Value     string            `json:"value,omitempty"`

	// The Hiro system that generated this event.
	System System `json:"-"`
	// Source ID represents the identifier of the event source, such as an achievement ID.
	SourceId string `json:"-"`
	// Source represents the configuration of the event source, such as an achievement config.
	Source any `json:"-"`
}

// The Publisher describes a service or similar target implementation that wishes to receive and process
// analytics-style events generated server-side by the various available Hiro systems.
//
// Each Publisher may choose to process or ignore each event as it sees fit. It may also choose to buffer
// events for batch processing at its discretion, but must take care to.
//
// Publisher implementations must safely handle concurrent calls.
//
// Implementations must handle any errors or retries internally, callers will not repeat calls in case
// of errors.
type Publisher interface {
	// Authenticate is called every time a user authenticates with Hiro. The 'created' flag is true if this
	// is a newly created user account, and each implementation may choose to handle this as it chooses.
	Authenticate(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, created bool)

	// Send is called when there are one or more events generated.
	Send(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, events []*PublisherEvent)
}
