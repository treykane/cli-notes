package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var benchmarkSearchIndexSink int

type benchmarkDataset struct {
	name       string
	mdCount    int
	txtCount   int
	dirFanout  int
	hitEvery   int
	queryToken string
}

func BenchmarkSearchIndex(b *testing.B) {
	datasets := []benchmarkDataset{
		{
			name:       "small",
			mdCount:    64,
			txtCount:   24,
			dirFanout:  4,
			hitEvery:   4,
			queryToken: "needle-small",
		},
		{
			name:       "large",
			mdCount:    2400,
			txtCount:   600,
			dirFanout:  12,
			hitEvery:   7,
			queryToken: "needle-large",
		},
	}

	for _, dataset := range datasets {
		dataset := dataset
		b.Run(dataset.name, func(b *testing.B) {
			root := b.TempDir()
			seedSearchBenchmarkDataset(b, root, dataset)

			b.Run("cold-build", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					idx := newSearchIndex(root)
					if err := idx.ensureBuilt(); err != nil {
						b.Fatalf("build index: %v", err)
					}
					benchmarkSearchIndexSink += len(idx.docs)
				}
			})

			b.Run("warm-query", func(b *testing.B) {
				idx := newSearchIndex(root)
				if err := idx.ensureBuilt(); err != nil {
					b.Fatalf("build index: %v", err)
				}

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					results := idx.search(dataset.queryToken)
					benchmarkSearchIndexSink += len(results)
				}
			})
		})
	}
}

func seedSearchBenchmarkDataset(b *testing.B, root string, dataset benchmarkDataset) {
	b.Helper()

	for i := 0; i < dataset.mdCount; i++ {
		dir := filepath.Join(root, fmt.Sprintf("folder-%02d", i%dataset.dirFanout))
		name := fmt.Sprintf("note-%04d.md", i)
		path := filepath.Join(dir, name)
		hitToken := ""
		if i%dataset.hitEvery == 0 {
			hitToken = dataset.queryToken
		}

		content := fmt.Sprintf("# Note %d\n\nProject log for benchmark dataset.\nToken: %s\n", i, hitToken)
		mustWriteBenchFile(b, path, content)
	}

	for i := 0; i < dataset.txtCount; i++ {
		dir := filepath.Join(root, fmt.Sprintf("assets-%02d", i%dataset.dirFanout))
		name := fmt.Sprintf("blob-%04d.txt", i)
		path := filepath.Join(dir, name)
		content := fmt.Sprintf("Text payload %d contains %s but must not match content search.\n", i, dataset.queryToken)
		mustWriteBenchFile(b, path, content)
	}
}

func mustWriteBenchFile(b *testing.B, path, content string) {
	b.Helper()

	if err := os.MkdirAll(filepath.Dir(path), DirPermission); err != nil {
		b.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), FilePermission); err != nil {
		b.Fatalf("write %q: %v", path, err)
	}
}
