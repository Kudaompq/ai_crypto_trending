package database

import (
	"errors"
	"fmt"
	"os"
)

// RemoveLegacyOpportunityDatabase removes the database used by the retired
// trading-opportunity feature.
func RemoveLegacyOpportunityDatabase(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove retired trading-opportunity database %q: %w", path, err)
	}
	return nil
}
