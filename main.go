package main

import (
	"context"
	"database/sql"
	"github.com/heroiclabs/nakama-common/runtime"
	"time"
)

// noinspection GoUnusedExportedFunction
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	initStart := time.Now()

	logger.Info("Loading Voidexforge Nakama plugin...")

	er := initializer.RegisterRpc("hello_world", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		return `{"message": "Hello, World!"}`, nil
	})
	if er != nil {
		logger.Error("Failed to register hello_world RPC: %v", er)
		return er
	}

	logger.Info("Voidexforge Nakama plugin loaded in '%d' msec.", time.Now().Sub(initStart).Milliseconds())
	return nil
}
