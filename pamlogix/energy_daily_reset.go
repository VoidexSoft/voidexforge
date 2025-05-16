package pamlogix

import (
	"context"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// ResetEnergyAtUTC resets a specific energy to its maximum value at a specific UTC time daily.
// This is useful for game mechanics like daily keys that reset at a specific time.
//
// When to call this method:
// - During user login/session start to ensure daily resets are checked
// - When the user tries to use an energy resource that follows a daily reset pattern
// - From a scheduled backend job that processes all users at the reset time
// - Anytime the client needs to validate if a daily-reset energy has been refreshed
//
// The method is idempotent - it will only perform the reset if needed based on the last reset time,
// so it's safe to call multiple times. It will not reset energy if the configured reset time
// hasn't been reached since the last reset.
//
// Parameters:
// - ctx: The context for the operation
// - logger: The Nakama logger
// - nk: The Nakama module
// - userID: The ID of the user whose energy is being reset
// - energyID: The ID of the energy to reset
// - hourUTC: The hour in UTC time when the reset should occur (0-23)
// - minuteUTC: The minute in UTC time when the reset should occur (0-59)
//
// Returns:
// - The updated energy value after the reset check
// - Any error that occurred during the process
func (e *NakamaEnergySystem) ResetEnergyAtUTC(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, energyID string, hourUTC int, minuteUTC int) (*Energy, error) {
	// Get all energies for the user
	energies, err := e.Get(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user energies: %v", err)
		return nil, err
	}

	// Check if the specific energy exists
	energy, exists := energies[energyID]
	if !exists {
		logger.Warn("Energy does not exist: %s", energyID)
		return nil, ErrBadInput
	}

	// Get the energy config
	_, exists = e.config.Energies[energyID]
	if !exists {
		logger.Warn("Energy config does not exist: %s", energyID)
		return nil, ErrBadInput
	}

	// Get current time in UTC
	now := time.Now().UTC()

	// Get the last reset time from additional properties
	lastResetProperty := "last_reset_time"
	lastResetTimeStr, exists := energy.AdditionalProperties[lastResetProperty]

	// Check if we need to perform a reset
	needsReset := false

	if !exists {
		// No reset has been done before, do it now
		needsReset = true
	} else {
		// Parse the last reset time
		lastResetTime, err := time.Parse(time.RFC3339, lastResetTimeStr)
		if err != nil {
			logger.Error("Failed to parse last reset time: %v", err)
			needsReset = true
		} else {
			// Calculate the next reset time
			nextResetTime := time.Date(
				lastResetTime.Year(),
				lastResetTime.Month(),
				lastResetTime.Day()+1, // Next day
				hourUTC,
				minuteUTC,
				0, 0,
				time.UTC,
			)

			// Check if we've passed the reset time
			if now.After(nextResetTime) {
				needsReset = true
			}
		}
	}

	if needsReset {
		// Reset the energy to max
		energy.Current = energy.Max

		// Update the last reset time
		if energy.AdditionalProperties == nil {
			energy.AdditionalProperties = make(map[string]string)
		}
		energy.AdditionalProperties[lastResetProperty] = now.Format(time.RFC3339)

		// Save the updated energy values
		energies[energyID] = energy
		if err := e.saveUserEnergies(ctx, logger, nk, userID, energies); err != nil {
			logger.Error("Failed to save user energies after reset: %v", err)
			return nil, ErrInternal
		}

		logger.Info("Reset energy %s for user %s", energyID, userID)
	}

	return energy, nil
}

// ResetTowerKeyDaily is a convenience method specifically for tower keys that reset at 0:00 UTC.
// This wraps the more general ResetEnergyAtUTC method with hour=0 and minute=0 for midnight UTC.
//
// When to call this method:
// - During user login/session start to ensure tower keys have been refreshed if needed
// - When a user attempts to access/use tower content requiring keys
// - From a scheduled job that runs daily after midnight UTC
// - After a user has spent all tower keys and needs to know when they'll reset
//
// Like ResetEnergyAtUTC, this method is idempotent and safe to call multiple times.
func (e *NakamaEnergySystem) ResetTowerKeyDaily(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, towerKeyID string) (*Energy, error) {
	return e.ResetEnergyAtUTC(ctx, logger, nk, userID, towerKeyID, 0, 0)
}

// GetWithDailyReset combines the functionality of Get with optional daily reset checking.
// It returns all energies for a user and optionally resets specific energies based on their daily reset times.
//
// Parameters:
//   - ctx: The context for the operation
//   - logger: The Nakama logger
//   - nk: The Nakama module
//   - userID: The ID of the user whose energies to get
//   - resetConfigs: Optional map of energy IDs to their reset times (hourUTC, minuteUTC)
//     If nil or empty, no resets will be performed (equivalent to just calling Get)
//
// Returns:
// - Map of all energies for the user, with any applicable resets already applied
// - Any error that occurred during the process
func (e *NakamaEnergySystem) GetWithDailyReset(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule,
	userID string, resetConfigs map[string][2]int) (map[string]*Energy, error) {

	// Get all energies for the user
	energies, err := e.Get(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user energies: %v", err)
		return nil, err
	}

	// Skip reset checks if no reset configs provided
	if resetConfigs == nil || len(resetConfigs) == 0 {
		return energies, nil
	}

	// Keep track if we need to save changes
	needsSave := false

	// Process each reset config
	now := time.Now().UTC()
	lastResetProperty := "last_reset_time"

	for energyID, resetTime := range resetConfigs {
		hourUTC, minuteUTC := resetTime[0], resetTime[1]

		// Check if the energy exists
		energy, exists := energies[energyID]
		if !exists {
			logger.Warn("Energy does not exist: %s", energyID)
			continue
		}

		// Check if the energy config exists
		if _, exists = e.config.Energies[energyID]; !exists {
			logger.Warn("Energy config does not exist: %s", energyID)
			continue
		}

		// Check if we need to reset this energy
		shouldReset := false
		lastResetTimeStr, exists := energy.AdditionalProperties[lastResetProperty]

		if !exists {
			// No reset has been done before, do it now
			shouldReset = true
		} else {
			// Parse the last reset time
			lastResetTime, err := time.Parse(time.RFC3339, lastResetTimeStr)
			if err != nil {
				logger.Error("Failed to parse last reset time: %v", err)
				shouldReset = true
			} else {
				// Calculate the next reset time
				nextResetTime := time.Date(
					lastResetTime.Year(),
					lastResetTime.Month(),
					lastResetTime.Day()+1, // Next day
					hourUTC,
					minuteUTC,
					0, 0,
					time.UTC,
				)

				// Check if we've passed the reset time
				if now.After(nextResetTime) {
					shouldReset = true
				}
			}
		}

		if shouldReset {
			// Reset the energy to max
			energy.Current = energy.Max

			// Update the last reset time
			if energy.AdditionalProperties == nil {
				energy.AdditionalProperties = make(map[string]string)
			}
			energy.AdditionalProperties[lastResetProperty] = now.Format(time.RFC3339)

			logger.Info("Reset energy %s for user %s", energyID, userID)
			needsSave = true
		}
	}

	// Save changes if needed
	if needsSave {
		if err := e.saveUserEnergies(ctx, logger, nk, userID, energies); err != nil {
			logger.Error("Failed to save user energies after reset: %v", err)
			return nil, ErrInternal
		}
	}

	return energies, nil
}

// GetWithDailyResetDefaults is a convenience wrapper around GetWithDailyReset
// that uses predefined reset configurations, for example midnight UTC (0:00)
// for tower keys.
//
// Parameters:
// - ctx: The context for the operation
// - logger: The Nakama logger
// - nk: The Nakama module
// - userID: The ID of the user whose energies to get
// - defaultResetEnergyIDs: List of energy IDs to reset at midnight UTC (00:00)
//
// Returns:
// - Map of all energies for the user, with any applicable resets at midnight already applied
// - Any error that occurred during the process
func (e *NakamaEnergySystem) GetWithDailyResetDefaults(ctx context.Context, logger runtime.Logger,
	nk runtime.NakamaModule, userID string, defaultResetEnergyIDs []string) (map[string]*Energy, error) {

	resetConfigs := make(map[string][2]int)
	for _, energyID := range defaultResetEnergyIDs {
		resetConfigs[energyID] = [2]int{0, 0} // Midnight UTC (00:00)
	}

	return e.GetWithDailyReset(ctx, logger, nk, userID, resetConfigs)
}
