package migrations

import "testing"

func TestRegistryIsOrderedAndUnique(t *testing.T) {
	seen := make(map[int64]bool)
	var previous int64
	for _, migration := range registry {
		if migration.Version <= previous {
			t.Fatalf("migration versions are not strictly increasing")
		}
		if seen[migration.Version] {
			t.Fatalf("duplicate migration version %d", migration.Version)
		}
		if migration.Name == "" || migration.Up == nil || migration.Down == nil {
			t.Fatalf("migration %d is incomplete", migration.Version)
		}
		seen[migration.Version] = true
		previous = migration.Version
	}
}
