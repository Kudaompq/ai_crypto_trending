package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveLegacyOpportunityDatabaseDeletesStoredData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opportunities.db")
	if err := os.WriteFile(path, []byte("legacy opportunities database"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveLegacyOpportunityDatabase(path); err != nil {
		t.Fatalf("remove legacy database: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy opportunity database still exists, stat error=%v", err)
	}
}

func TestRemoveLegacyOpportunityDatabaseAllowsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-opportunities.db")
	if err := RemoveLegacyOpportunityDatabase(path); err != nil {
		t.Fatalf("remove missing legacy database: %v", err)
	}
}

func TestRemoveLegacyOpportunityDatabaseReportsRemovalFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonempty-directory")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "keep"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveLegacyOpportunityDatabase(path); err == nil {
		t.Fatal("expected an error when the legacy database path cannot be removed")
	}
}
