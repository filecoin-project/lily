package vector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteVectors(t *testing.T) {
	t.Skipf("skipping, see https://github.com/filecoin-project/sentinel-visor/issues/603")
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
		t.Run(testName(vpath), func(t *testing.T) {
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
		b.Run(testName(vpath), func(b *testing.B) {
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

func testName(vpath string) string {
	name := filepath.Base(vpath)

	ext := filepath.Ext(name)

	if len(ext) > 0 {
		name = name[:len(name)-len(ext)]
	}

	if strings.HasPrefix(name, "Qm") {
		idx := strings.Index(name, "_")
		if idx > 0 {
			name = name[idx+1:]
		}
	}

	return name
}
