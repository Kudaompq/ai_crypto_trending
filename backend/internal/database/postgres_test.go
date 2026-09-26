package database

import (
	"context"
	"strings"
	"testing"
)

func TestOpenPostgresRequiresConnectionURL(t *testing.T) {
	_, err := OpenPostgres(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "WATCHLIST_DATABASE_URL is required") {
		t.Fatalf("OpenPostgres error = %v, want required connection URL", err)
	}
}
