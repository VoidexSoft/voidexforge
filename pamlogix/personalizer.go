package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

// The Personalizer describes an intermediate server or service which can be used to personalize the base data
// definitions defined for the gameplay systems.
type Personalizer interface {
	// GetValue returns a config which has been modified for a gameplay system,
	// or nil if the config is not being adjusted by this personalizer.
	GetValue(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, system System, identity string) (config any, err error)
}
