package pamlogix

import (
	"context"
	"testing"

	"github.com/heroiclabs/nakama-common/runtime"
)

func TestTutorialsSystem_GetType(t *testing.T) {
	config := &TutorialsConfig{
		Tutorials: map[string]*TutorialsConfigTutorial{
			"test_tutorial": {
				StartStep: 0,
				MaxStep:   5,
			},
		},
	}

	system := NewNakamaTutorialsSystem(config)
	if system.GetType() != SystemTypeTutorials {
		t.Errorf("Expected system type %v, got %v", SystemTypeTutorials, system.GetType())
	}
}

func TestTutorialsSystem_GetConfig(t *testing.T) {
	config := &TutorialsConfig{
		Tutorials: map[string]*TutorialsConfigTutorial{
			"test_tutorial": {
				StartStep: 0,
				MaxStep:   5,
			},
		},
	}

	system := NewNakamaTutorialsSystem(config)
	returnedConfig := system.GetConfig().(*TutorialsConfig)
	if returnedConfig != config {
		t.Error("GetConfig should return the same config instance")
	}
}

func TestTutorialsSystem_SetOnStepCompleted(t *testing.T) {
	config := &TutorialsConfig{
		Tutorials: map[string]*TutorialsConfigTutorial{
			"test_tutorial": {
				StartStep: 0,
				MaxStep:   5,
			},
		},
	}

	system := NewNakamaTutorialsSystem(config)

	// Test that SetOnStepCompleted doesn't panic and can be called
	system.SetOnStepCompleted(func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, config *TutorialsConfigTutorial, resetCount, step int, prevStep *int) {
		// This callback is just for testing that it can be set
	})

	// Test that we can set it to nil as well
	system.SetOnStepCompleted(nil)

	// Test that we can set it again
	system.SetOnStepCompleted(func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, tutorialID string, config *TutorialsConfigTutorial, resetCount, step int, prevStep *int) {
		// Another callback
	})

	// If we reach here without panicking, the test passes
}

func TestTutorialsConfigTutorial_Validation(t *testing.T) {
	tutorial := &TutorialsConfigTutorial{
		StartStep:            0,
		MaxStep:              5,
		AdditionalProperties: map[string]string{"category": "test"},
	}

	if tutorial.StartStep != 0 {
		t.Errorf("Expected StartStep 0, got %d", tutorial.StartStep)
	}

	if tutorial.MaxStep != 5 {
		t.Errorf("Expected MaxStep 5, got %d", tutorial.MaxStep)
	}

	if tutorial.AdditionalProperties["category"] != "test" {
		t.Errorf("Expected category 'test', got '%s'", tutorial.AdditionalProperties["category"])
	}
}
