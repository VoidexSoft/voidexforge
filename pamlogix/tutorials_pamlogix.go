package pamlogix

import (
	"context"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	tutorialsStorageCollection = "tutorials"
	userTutorialsStorageKey    = "user_tutorials"
)

// NakamaTutorialsSystem implements the TutorialsSystem interface using Nakama as the backend.
type NakamaTutorialsSystem struct {
	config          *TutorialsConfig
	pamlogix        Pamlogix
	onStepCompleted func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, config *TutorialsConfigTutorial, resetCount, step int, prevStep *int)
}

// NewNakamaTutorialsSystem creates a new instance of the tutorials system with the given configuration.
func NewNakamaTutorialsSystem(config *TutorialsConfig) *NakamaTutorialsSystem {
	return &NakamaTutorialsSystem{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this tutorials system
func (t *NakamaTutorialsSystem) SetPamlogix(pl Pamlogix) {
	t.pamlogix = pl
}

// GetType returns the system type for the tutorials system.
func (t *NakamaTutorialsSystem) GetType() SystemType {
	return SystemTypeTutorials
}

// GetConfig returns the configuration for the tutorials system.
func (t *NakamaTutorialsSystem) GetConfig() any {
	return t.config
}

// Get returns all tutorials defined and progress made by the user towards them.
func (t *NakamaTutorialsSystem) Get(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (tutorials map[string]*Tutorial, err error) {
	// Get user's tutorial progress from storage
	userTutorials, err := t.getUserTutorials(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user tutorials: %v", err)
		return nil, err
	}

	// Initialize result map
	tutorials = make(map[string]*Tutorial)

	// Create tutorials based on config and merge with user progress
	for tutorialID, tutorialConfig := range t.config.Tutorials {
		tutorial := &Tutorial{
			Id:                   tutorialID,
			Current:              int32(tutorialConfig.StartStep),
			Max:                  int32(tutorialConfig.MaxStep),
			State:                TutorialState_TUTORIAL_STATE_NONE,
			UpdateTimeSec:        time.Now().Unix(),
			CompleteTimeSec:      0,
			AdditionalProperties: tutorialConfig.AdditionalProperties,
		}

		// Merge with user progress if it exists
		if userTutorial, exists := userTutorials[tutorialID]; exists {
			tutorial.Current = userTutorial.Current
			tutorial.State = userTutorial.State
			tutorial.UpdateTimeSec = userTutorial.UpdateTimeSec
			tutorial.CompleteTimeSec = userTutorial.CompleteTimeSec
			// Merge additional properties
			if userTutorial.AdditionalProperties != nil {
				if tutorial.AdditionalProperties == nil {
					tutorial.AdditionalProperties = make(map[string]string)
				}
				for k, v := range userTutorial.AdditionalProperties {
					tutorial.AdditionalProperties[k] = v
				}
			}
		}

		tutorials[tutorialID] = tutorial
	}

	return tutorials, nil
}

// Accept marks a tutorial as accepted by the user.
func (t *NakamaTutorialsSystem) Accept(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, tutorialID string, userID string) (tutorial *Tutorial, err error) {
	// Check if tutorial exists in config
	tutorialConfig, exists := t.config.Tutorials[tutorialID]
	if !exists {
		return nil, runtime.NewError("tutorial not found", NOT_FOUND_ERROR_CODE) // NOT_FOUND
	}

	// Get current user tutorials
	userTutorials, err := t.getUserTutorials(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Create or update tutorial
	tutorial = &Tutorial{
		Id:                   tutorialID,
		Current:              int32(tutorialConfig.StartStep),
		Max:                  int32(tutorialConfig.MaxStep),
		State:                TutorialState_TUTORIAL_STATE_ACCEPTED,
		UpdateTimeSec:        time.Now().Unix(),
		CompleteTimeSec:      0,
		AdditionalProperties: tutorialConfig.AdditionalProperties,
	}

	// If tutorial already exists, preserve some fields
	if existingTutorial, exists := userTutorials[tutorialID]; exists {
		// Only allow accepting if not already completed
		if existingTutorial.State == TutorialState_TUTORIAL_STATE_COMPLETED {
			return nil, runtime.NewError("tutorial already completed", FAILED_PRECONDITION_ERROR_CODE) // FAILED_PRECONDITION
		}
		// Preserve current step if it's higher than start step
		if existingTutorial.Current > int32(tutorialConfig.StartStep) {
			tutorial.Current = existingTutorial.Current
			tutorial.State = TutorialState_TUTORIAL_STATE_IN_PROGRESS
		}
	}

	// Update user tutorials
	userTutorials[tutorialID] = tutorial

	// Save to storage
	err = t.saveUserTutorials(ctx, logger, nk, userID, userTutorials)
	if err != nil {
		return nil, err
	}

	return tutorial, nil
}

// Decline marks a tutorial as declined by the user.
func (t *NakamaTutorialsSystem) Decline(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, tutorialID string, userID string) (tutorial *Tutorial, err error) {
	// Check if tutorial exists in config
	tutorialConfig, exists := t.config.Tutorials[tutorialID]
	if !exists {
		return nil, runtime.NewError("tutorial not found", NOT_FOUND_ERROR_CODE) // NOT_FOUND
	}

	// Get current user tutorials
	userTutorials, err := t.getUserTutorials(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Create or update tutorial
	tutorial = &Tutorial{
		Id:                   tutorialID,
		Current:              int32(tutorialConfig.StartStep),
		Max:                  int32(tutorialConfig.MaxStep),
		State:                TutorialState_TUTORIAL_STATE_DECLINED,
		UpdateTimeSec:        time.Now().Unix(),
		CompleteTimeSec:      0,
		AdditionalProperties: tutorialConfig.AdditionalProperties,
	}

	// If tutorial already exists, preserve current step
	if existingTutorial, exists := userTutorials[tutorialID]; exists {
		tutorial.Current = existingTutorial.Current
	}

	// Update user tutorials
	userTutorials[tutorialID] = tutorial

	// Save to storage
	err = t.saveUserTutorials(ctx, logger, nk, userID, userTutorials)
	if err != nil {
		return nil, err
	}

	return tutorial, nil
}

// Abandon marks the tutorial as abandoned by the user.
func (t *NakamaTutorialsSystem) Abandon(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, tutorialID string, userID string) (tutorial *Tutorial, err error) {
	// Check if tutorial exists in config
	tutorialConfig, exists := t.config.Tutorials[tutorialID]
	if !exists {
		return nil, runtime.NewError("tutorial not found", NOT_FOUND_ERROR_CODE) // NOT_FOUND
	}

	// Get current user tutorials
	userTutorials, err := t.getUserTutorials(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Create or update tutorial
	tutorial = &Tutorial{
		Id:                   tutorialID,
		Current:              int32(tutorialConfig.StartStep),
		Max:                  int32(tutorialConfig.MaxStep),
		State:                TutorialState_TUTORIAL_STATE_ABANDONED,
		UpdateTimeSec:        time.Now().Unix(),
		CompleteTimeSec:      0,
		AdditionalProperties: tutorialConfig.AdditionalProperties,
	}

	// If tutorial already exists, preserve current step
	if existingTutorial, exists := userTutorials[tutorialID]; exists {
		tutorial.Current = existingTutorial.Current
	}

	// Update user tutorials
	userTutorials[tutorialID] = tutorial

	// Save to storage
	err = t.saveUserTutorials(ctx, logger, nk, userID, userTutorials)
	if err != nil {
		return nil, err
	}

	return tutorial, nil
}

// Update modifies a tutorial by its ID to step through it for the user by ID.
func (t *NakamaTutorialsSystem) Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, step int) (tutorials map[string]*Tutorial, err error) {
	// Check if tutorial exists in config
	tutorialConfig, exists := t.config.Tutorials[tutorialID]
	if !exists {
		return nil, runtime.NewError("tutorial not found", NOT_FOUND_ERROR_CODE) // NOT_FOUND
	}

	// Validate step
	if step < tutorialConfig.StartStep || step > tutorialConfig.MaxStep {
		return nil, runtime.NewError("invalid step", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
	}

	// Get current user tutorials
	userTutorials, err := t.getUserTutorials(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Get or create tutorial
	tutorial, exists := userTutorials[tutorialID]
	if !exists {
		// Auto-accept tutorial if it doesn't exist
		tutorial = &Tutorial{
			Id:                   tutorialID,
			Current:              int32(tutorialConfig.StartStep),
			Max:                  int32(tutorialConfig.MaxStep),
			State:                TutorialState_TUTORIAL_STATE_ACCEPTED,
			UpdateTimeSec:        time.Now().Unix(),
			CompleteTimeSec:      0,
			AdditionalProperties: tutorialConfig.AdditionalProperties,
		}
	}

	// Store previous step for callback
	var prevStep *int
	if tutorial.Current != int32(step) {
		prev := int(tutorial.Current)
		prevStep = &prev
	}

	// Update tutorial progress
	tutorial.Current = int32(step)
	tutorial.UpdateTimeSec = time.Now().Unix()

	// Update state based on progress
	if step >= tutorialConfig.MaxStep {
		tutorial.State = TutorialState_TUTORIAL_STATE_COMPLETED
		tutorial.CompleteTimeSec = time.Now().Unix()
	} else if tutorial.State == TutorialState_TUTORIAL_STATE_ACCEPTED || tutorial.State == TutorialState_TUTORIAL_STATE_NONE {
		tutorial.State = TutorialState_TUTORIAL_STATE_IN_PROGRESS
	}

	// Update user tutorials
	userTutorials[tutorialID] = tutorial

	// Save to storage
	err = t.saveUserTutorials(ctx, logger, nk, userID, userTutorials)
	if err != nil {
		return nil, err
	}

	// Call step completed callback if set
	if t.onStepCompleted != nil {
		// Get reset count (for now, we'll use 0 as we don't track resets yet)
		resetCount := 0
		t.onStepCompleted(ctx, logger, nk, userID, tutorialID, tutorialConfig, resetCount, step, prevStep)
	}

	// Return all tutorials
	return t.Get(ctx, logger, nk, userID)
}

// Reset wipes all known state for the given tutorial identifier(s).
func (t *NakamaTutorialsSystem) Reset(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, tutorialIDs []string) (tutorials map[string]*Tutorial, err error) {
	// Get current user tutorials
	userTutorials, err := t.getUserTutorials(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Reset specified tutorials
	for _, tutorialID := range tutorialIDs {
		// Check if tutorial exists in config
		tutorialConfig, exists := t.config.Tutorials[tutorialID]
		if !exists {
			logger.Warn("Tutorial %s not found in config, skipping reset", tutorialID)
			continue
		}

		// Reset tutorial to initial state
		userTutorials[tutorialID] = &Tutorial{
			Id:                   tutorialID,
			Current:              int32(tutorialConfig.StartStep),
			Max:                  int32(tutorialConfig.MaxStep),
			State:                TutorialState_TUTORIAL_STATE_NONE,
			UpdateTimeSec:        time.Now().Unix(),
			CompleteTimeSec:      0,
			AdditionalProperties: tutorialConfig.AdditionalProperties,
		}
	}

	// Save to storage
	err = t.saveUserTutorials(ctx, logger, nk, userID, userTutorials)
	if err != nil {
		return nil, err
	}

	// Return all tutorials
	return t.Get(ctx, logger, nk, userID)
}

// SetOnStepCompleted registers a hook that fires on tutorial step completions.
func (t *NakamaTutorialsSystem) SetOnStepCompleted(fn func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, config *TutorialsConfigTutorial, resetCount, step int, prevStep *int)) {
	t.onStepCompleted = fn
}

// getUserTutorials retrieves the user's tutorial progress from storage
func (t *NakamaTutorialsSystem) getUserTutorials(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*Tutorial, error) {
	collection := tutorialsStorageCollection
	if t.pamlogix != nil && t.pamlogix.(*pamlogixImpl).collectionResolver != nil {
		resolvedCollection, err := t.pamlogix.(*pamlogixImpl).collectionResolver(ctx, SystemTypeTutorials, collection)
		if err != nil {
			logger.Warn("Failed to resolve collection name: %v", err)
		} else {
			collection = resolvedCollection
		}
	}

	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: collection,
			Key:        userTutorialsStorageKey,
			UserID:     userID,
		},
	})
	if err != nil {
		logger.Error("Failed to read user tutorials from storage: %v", err)
		return nil, runtime.NewError("failed to read user tutorials", INTERNAL_ERROR_CODE) // INTERNAL
	}

	userTutorials := make(map[string]*Tutorial)
	if len(objects) > 0 && objects[0].Value != "" {
		if err := json.Unmarshal([]byte(objects[0].Value), &userTutorials); err != nil {
			logger.Error("Failed to unmarshal user tutorials: %v", err)
			return nil, runtime.NewError("failed to unmarshal user tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}
	}

	return userTutorials, nil
}

// saveUserTutorials saves the user's tutorial progress to storage
func (t *NakamaTutorialsSystem) saveUserTutorials(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, userTutorials map[string]*Tutorial) error {
	collection := tutorialsStorageCollection
	if t.pamlogix != nil && t.pamlogix.(*pamlogixImpl).collectionResolver != nil {
		resolvedCollection, err := t.pamlogix.(*pamlogixImpl).collectionResolver(ctx, SystemTypeTutorials, collection)
		if err != nil {
			logger.Warn("Failed to resolve collection name: %v", err)
		} else {
			collection = resolvedCollection
		}
	}

	data, err := json.Marshal(userTutorials)
	if err != nil {
		logger.Error("Failed to marshal user tutorials: %v", err)
		return runtime.NewError("failed to marshal user tutorials", INTERNAL_ERROR_CODE) // INTERNAL
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: collection,
			Key:        userTutorialsStorageKey,
			UserID:     userID,
			Value:      string(data),
		},
	})
	if err != nil {
		logger.Error("Failed to write user tutorials to storage: %v", err)
		return runtime.NewError("failed to write user tutorials", INTERNAL_ERROR_CODE) // INTERNAL
	}

	return nil
}
