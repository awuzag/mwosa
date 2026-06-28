package mongodb

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepositoryPackagesHaveMongoDBImplementationsOrDocumentedExclusions(t *testing.T) {
	excluded := map[string]string{
		"migration":    "legacy SQL migration run history is not migrated into canonical MongoDB storage",
		"providerauth": "provider auth token cache stays separate from canonical MongoDB runtime",
	}

	storageDir := filepath.Clean("..")
	entries, err := os.ReadDir(storageDir)
	require.NoError(t, err)

	var missing []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		repositoryPath := filepath.Join(storageDir, name, "repository.go")
		if _, err := os.Stat(repositoryPath); err != nil {
			continue
		}
		mongoPath := filepath.Join(storageDir, name, "mongodb_repository.go")
		_, mongoErr := os.Stat(mongoPath)
		_, isExcluded := excluded[name]
		switch {
		case mongoErr == nil && isExcluded:
			missing = append(missing, name+" has a MongoDB implementation but is still listed as excluded")
		case mongoErr != nil && !isExcluded:
			missing = append(missing, name+" is missing mongodb_repository.go")
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing)
}

func TestCanonicalMongoDBCollectionsExcludeLegacySidecars(t *testing.T) {
	names := collectionNames(CollectionSpecs())

	require.NotContains(t, names, "migration_runs")
	require.NotContains(t, names, "provider_auth_tokens")
}
