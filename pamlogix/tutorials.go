package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

// TutorialsConfig is the data definition for the TutorialsSystem type.
type TutorialsConfig struct {
	Tutorials map[string]*TutorialsConfigTutorial `json:"tutorials,omitempty"`
}

type TutorialsConfigTutorial struct {
	StartStep            int               `json:"start_step,omitempty"`
	MaxStep              int               `json:"max_step,omitempty"`
	AdditionalProperties map[string]string `json:"additional_properties,omitempty"`
}

// The TutorialsSystem is a gameplay system which records progress made through tutorials.
type TutorialsSystem interface {
	System

	// Get returns all tutorials defined and progress made by the user towards them.
	Get(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (tutorials map[string]*Tutorial, err error)

	// Accept marks a tutorial as accepted by the user.
	Accept(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, tutorialID string, userID string) (tutorial *Tutorial, err error)

	// Decline marks a tutorial as declined by the user.
	Decline(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, tutorialID string, userID string) (tutorial *Tutorial, err error)

	// Abandon marks the tutorial as abandoned by the user.
	Abandon(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, tutorialID string, userID string) (tutorial *Tutorial, err error)

	// Update modifies a tutorial by its ID to step through it for the user by ID.
	Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, step int) (tutorial map[string]*Tutorial, err error)

	// Reset wipes all known state for the given tutorial identifier(s).
	Reset(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, tutorialIDs []string) (tutorials map[string]*Tutorial, err error)

	// SetOnStepCompleted registers a hook that fires on tutorial step completions.
	SetOnStepCompleted(func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, config *TutorialsConfigTutorial, resetCount, step int, prevStep *int))
}
