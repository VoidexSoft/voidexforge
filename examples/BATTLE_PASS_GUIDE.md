# Battle Pass Implementation Guide

## Overview

Yes! The progression system is perfectly designed for implementing battle pass systems. With the recent improvements, it now supports all the advanced features needed for sophisticated battle pass mechanics.

## Key Features for Battle Pass

### ✅ **Tier-Based Progression**
- Use `counts` for XP requirements
- Use `progressions` dependencies for sequential unlocking
- Use `additional_properties` for metadata (tier, season, pass type)

### ✅ **Free vs Premium Tracks**
- Use `items_min` to require premium pass purchase
- Different reward tiers based on pass ownership
- Logical operators for complex unlock conditions

### ✅ **Seasonal Resets**
- Use `reset_schedule` with CRON expressions
- Weekly/daily challenges with automatic resets
- Season-based progression tracking

### ✅ **Rich Rewards System**
- Guaranteed rewards for each tier
- Currency and item rewards
- Cosmetic items, weapons, and exclusive content

### ✅ **Complex Requirements**
- Achievement-based unlocks
- Player level requirements
- Multiple challenge completion
- Logical operators (AND/OR/XOR) for advanced conditions

## Battle Pass Architecture

### 1. **Tier Structure**

Each battle pass tier is a separate progression:

```json
{
  "battle_pass_tier_X": {
    "name": "Battle Pass Tier X",
    "category": "battle_pass",
    "additional_properties": {
      "tier": "X",
      "season": "winter_2024",
      "pass_type": "free|premium",
      "xp_required": "amount"
    },
    "preconditions": {
      "direct": {
        "counts": {"battle_pass_xp": X},
        "progressions": ["previous_tier"]
      }
    },
    "rewards": {
      "guaranteed": {
        "currencies": {"coins": {"min": X, "max": X}},
        "items": {"reward_item": {"min": 1, "max": 1}}
      }
    }
  }
}
```

### 2. **Premium vs Free Tracks**

**Free Track**: No additional requirements
```json
{
  "preconditions": {
    "direct": {
      "counts": {"battle_pass_xp": 100}
    }
  }
}
```

**Premium Track**: Requires premium pass purchase
```json
{
  "preconditions": {
    "direct": {
      "counts": {"battle_pass_xp": 100},
      "items_min": {"battle_pass_premium": 1}
    }
  }
}
```

### 3. **Challenge System**

**Daily Challenges** (Reset every day):
```json
{
  "battle_pass_daily_login": {
    "preconditions": {
      "direct": {
        "counts": {"daily_logins": 1}
      }
    },
    "reset_schedule": "0 0 * * *",
    "rewards": {
      "guaranteed": {
        "currencies": {"battle_pass_xp": {"min": 50, "max": 50}}
      }
    }
  }
}
```

**Weekly Challenges** (Reset every Monday):
```json
{
  "battle_pass_weekly_challenge": {
    "preconditions": {
      "direct": {
        "counts": {"weekly_challenges_completed": 3}
      }
    },
    "reset_schedule": "0 0 * * 1",
    "rewards": {
      "guaranteed": {
        "currencies": {"battle_pass_xp": {"min": 500, "max": 500}}
      }
    }
  }
}
```

### 4. **Exclusive Tiers**

Use logical operators for special requirements:

```json
{
  "battle_pass_tier_exclusive": {
    "preconditions": {
      "direct": {
        "counts": {"battle_pass_xp": 1000},
        "items_min": {"battle_pass_premium": 1}
      },
      "operator": 1,
      "nested": {
        "direct": {
          "achievements": ["season_champion"]
        },
        "operator": 2,
        "nested": {
          "direct": {
            "stats_min": {"player_level": 50}
          }
        }
      }
    }
  }
}
```

This means: **(XP + Premium) AND (Achievement OR Level 50)**

## Implementation Workflow

### 1. **Setup Battle Pass Season**

```go
// Initialize progression system with battle pass config
progressionSystem := NewNakamaProgressionSystem(battlePassConfig)

// Grant premium pass to user (via purchase)
economySystem.RewardGrant(ctx, logger, nk, userID, &Reward{
    Items: map[string]int64{"battle_pass_premium": 1}
}, metadata, false)
```

### 2. **Award Battle Pass XP**

```go
// When player completes activities, award XP
progressionSystem.Update(ctx, logger, nk, userID, "battle_pass_xp_tracker", map[string]int64{
    "battle_pass_xp": 25  // Award 25 XP
})
```

### 3. **Check Tier Unlocks**

```go
// Get current progression state
progressions, deltas, err := progressionSystem.Get(ctx, logger, nk, userID, lastKnownState)

// Check which tiers are newly unlocked
for tierID, delta := range deltas {
    if delta.State == ProgressionDeltaState_PROGRESSION_DELTA_STATE_UNLOCKED {
        // Tier unlocked! Grant rewards automatically
        progressionSystem.Complete(ctx, logger, nk, userID, tierID)
    }
}
```

### 4. **Handle Season Reset**

```go
// At season end, reset all battle pass progressions
battlePassTiers := []string{
    "battle_pass_tier_1", "battle_pass_tier_2", 
    "battle_pass_tier_3_premium", // ... etc
}

progressionSystem.Reset(ctx, logger, nk, userID, battlePassTiers)
```

## Advanced Features

### 1. **Retroactive Premium Rewards**

When a player purchases premium mid-season:

```go
// Check all completed free tiers
progressions, _, _ := progressionSystem.Get(ctx, logger, nk, userID, nil)

for tierID, progression := range progressions {
    if progression.Unlocked && isPremiumTier(tierID) {
        // Grant premium rewards retroactively
        progressionSystem.Complete(ctx, logger, nk, userID, tierID)
    }
}
```

### 2. **XP Boosters**

```json
{
  "xp_booster_weekend": {
    "preconditions": {
      "direct": {
        "items_min": {"weekend_booster": 1}
      }
    },
    "rewards": {
      "guaranteed": {
        "currencies": {"battle_pass_xp": {"min": 100, "max": 100}}
      }
    }
  }
}
```

### 3. **Milestone Rewards**

```json
{
  "battle_pass_milestone_25": {
    "name": "25% Completion Milestone",
    "preconditions": {
      "direct": {
        "counts": {"battle_pass_xp": 1250}
      }
    },
    "rewards": {
      "guaranteed": {
        "items": {"milestone_chest": {"min": 1, "max": 1}}
      }
    }
  }
}
```

## Benefits of Using Progression System

### ✅ **Automatic Validation**
- XP requirements automatically checked
- Premium pass ownership validated
- Sequential tier unlocking enforced

### ✅ **Flexible Requirements**
- Combine XP, achievements, player level, items
- Use logical operators for complex conditions
- Support both free and premium tracks

### ✅ **Built-in Rewards**
- Automatic reward granting on completion
- Integration with economy system
- Support for currencies, items, and cosmetics

### ✅ **Seasonal Management**
- CRON-based reset schedules
- Daily/weekly challenge resets
- Season-end cleanup

### ✅ **Real-time Updates**
- Delta tracking for UI updates
- Immediate unlock notifications
- Progress synchronization

## Example API Usage

```go
// Award XP for completing a match
progressionSystem.Update(ctx, logger, nk, userID, "battle_pass_xp_tracker", 
    map[string]int64{"battle_pass_xp": 50})

// Check for tier unlocks
progressions, deltas, _ := progressionSystem.Get(ctx, logger, nk, userID, lastKnown)

// Complete daily challenge
progressionSystem.Complete(ctx, logger, nk, userID, "battle_pass_daily_login")

// Purchase premium pass
economySystem.RewardGrant(ctx, logger, nk, userID, &Reward{
    Items: map[string]int64{"battle_pass_premium": 1}
}, metadata, false)

// Reset for new season
progressionSystem.Reset(ctx, logger, nk, userID, allBattlePassTiers)
```

The progression system provides everything needed for a full-featured battle pass implementation with minimal additional code! 