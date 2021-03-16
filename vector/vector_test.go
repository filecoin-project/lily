package vector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteVectors(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping, short testing specified")
	}
	ctx := context.Background()
	var vectorPaths []string
	if err := filepath.Walk("./data", func(path string, info os.FileInfo, _ error) error {
		if filepath.Ext(path) != ".json" {
			return nil
		}
		full, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}
		vectorPaths = append(vectorPaths, full)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	for _, vp := range vectorPaths {
		vpath := vp
		t.Run(filepath.Base(vpath), func(t *testing.T) {
			runner, err := NewRunner(ctx, vpath, 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := runner.Run(ctx); err != nil {
				t.Fatal(err)
			}
			if err := runner.Validate(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func BenchmarkExecuteVectors(b *testing.B) {
	if testing.Short() {
		b.Skipf("skipping, short testing specified")
	}

	ctx := context.Background()
	var vectorPaths []string
	if err := filepath.Walk("./data", func(path string, info os.FileInfo, _ error) error {
		if filepath.Ext(path) != ".json" {
			return nil
		}
		full, err := filepath.Abs(path)
		if err != nil {
			b.Fatal(err)
		}
		vectorPaths = append(vectorPaths, full)
		return nil
	}); err != nil {
		b.Fatal(err)
	}

	for _, vp := range vectorPaths {
		vpath := vp
		b.Run(filepath.Base(vpath), func(b *testing.B) {
			runner, err := NewRunner(ctx, vpath, 0)
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := runner.Run(ctx); err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(runner.BlockstoreGetCount())/float64(b.N), "gets/op")
				runner.Reset()
			}
		})
	}
}
