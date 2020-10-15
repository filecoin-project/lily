package migrations

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/go-pg/migrations/v8"
)

var (
	deferredMu sync.Mutex
	deferred   = map[int64]Migration{}
)

type Migration struct {
	Version  int64
	Up, Down func(migrations.DB) error
}

// MustRegisterDeferred registers a deferred migration. The up and down functions should be idempotent.
// Only one deferred migration may be registered for each schema version.
func MustRegisterDeferred(up, down func(migrations.DB) error) {
	deferredMu.Lock()
	defer deferredMu.Unlock()

	version, err := extractVersionGo(migrationFile())
	if err != nil {
		panic(err.Error())
	}

	if _, ok := deferred[version]; ok {
		panic(fmt.Errorf("Attempt to register a second deferred migration for schema version %d", version))
	}

	deferred[version] = Migration{
		Version: version,
		Up:      up,
		Down:    down,
	}
}

// DeferredMigrations returns a map of deferred migrations by schema version.
func DeferredMigrations() map[int64]Migration {
	dms := make(map[int64]Migration, len(deferred))
	for v, m := range deferred {
		dms[v] = m
	}
	return dms
}

func migrationFile() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:]) // skip current and caller frames
	frames := runtime.CallersFrames(pcs[:n])

	for {
		f, ok := frames.Next()
		if !ok {
			break
		}
		if !strings.Contains(f.Function, "RegisterDeferred") {
			return f.File
		}
	}

	return ""
}

func extractVersionGo(name string) (int64, error) {
	base := filepath.Base(name)
	if !strings.HasSuffix(name, ".go") {
		return 0, fmt.Errorf("file=%q must have extension .go", base)
	}

	idx := strings.IndexByte(base, '_')
	if idx == -1 {
		err := fmt.Errorf(
			"file=%q must have name in format version_comment, e.g. 1_initial",
			base)
		return 0, err
	}

	n, err := strconv.ParseInt(base[:idx], 10, 64)
	if err != nil {
		return 0, err
	}

	return n, nil
}
