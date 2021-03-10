package vector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteVectors(t *testing.T) {
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
