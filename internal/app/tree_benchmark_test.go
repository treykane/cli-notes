package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var benchmarkTreeRebuildSink int

type treeBenchmarkDataset struct {
	name    string
	mdCount int
	fanout  int
}

func BenchmarkTreeRebuild(b *testing.B) {
	datasets := []treeBenchmarkDataset{
		{name: "medium", mdCount: 1200, fanout: 12},
		{name: "large", mdCount: 6000, fanout: 32},
	}

	for _, dataset := range datasets {
		dataset := dataset
		b.Run(dataset.name, func(b *testing.B) {
			root := b.TempDir()
			seedTreeBenchmarkDataset(b, root, dataset)

			m := &Model{
				notesDir:          root,
				expanded:          map[string]bool{root: true},
				sortMode:          sortModeName,
				pinnedPaths:       map[string]bool{},
				treeMetadataCache: map[string]treeMetadataCacheEntry{},
			}

			keepPath := filepath.Join(root, "group-00", "note-0000.md")
			m.expandParentDirs(keepPath)

			b.Run("cold-cache-rebuild", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					m.invalidateTreeMetadataCache()
					m.rebuildTreeKeep(keepPath)
					benchmarkTreeRebuildSink += len(m.items)
				}
			})

			b.Run("warm-cache-rebuild", func(b *testing.B) {
				m.rebuildTreeKeep(keepPath)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					m.rebuildTreeKeep(keepPath)
					benchmarkTreeRebuildSink += len(m.treeMetadataCache)
				}
			})
		})
	}
}

func seedTreeBenchmarkDataset(b *testing.B, root string, dataset treeBenchmarkDataset) {
	b.Helper()

	for i := 0; i < dataset.mdCount; i++ {
		dir := filepath.Join(root, fmt.Sprintf("group-%02d", i%dataset.fanout))
		path := filepath.Join(dir, fmt.Sprintf("note-%04d.md", i))
		content := fmt.Sprintf("---\ntitle: Note %d\ntags: [bench,tree]\n---\n\n# Note %d\n\nbenchmark content\n", i, i)
		if err := os.MkdirAll(filepath.Dir(path), DirPermission); err != nil {
			b.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), FilePermission); err != nil {
			b.Fatalf("write %q: %v", path, err)
		}
	}
}
