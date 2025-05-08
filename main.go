// Package main is the entry point for the Nakama plugin.
package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// InitModule is called by Nakama when the plugin is loaded.
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	initStart := time.Now()

	logger.Info("Loading Voidexforge Nakama plugin...")

	// Register an example RPC function
	if err := initializer.RegisterRpc("hello_world", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		return "{\"message\":\"Hello World!\"}", nil
	}); err != nil {
		return err
	}

	logger.Info("Voidexforge Nakama plugin loaded in '%d' msec.", time.Now().Sub(initStart).Milliseconds())
	return nil
}
