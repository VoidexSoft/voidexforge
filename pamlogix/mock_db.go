//go:build test
// +build test

package pamlogix

import (
	"database/sql"
	"os"
	"testing"
)

func NewDB(t *testing.T) *sql.DB {
	//dbUrl := "postgresql://postgres@127.0.0.1:5432/nakama?sslmode=disable"
	dbUrl := "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable"
	if dbUrlEnv := os.Getenv("TEST_DB_URL"); len(dbUrlEnv) > 0 {
		dbUrl = dbUrlEnv
	}

	db, err := sql.Open("pgx", dbUrl)
	if err != nil {
		t.Fatal("Error connecting to database", err)
	}
	err = db.Ping()
	if err != nil {
		t.Fatal("Error pinging database", err)
	}
	return db
}
