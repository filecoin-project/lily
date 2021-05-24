package model

import (
	"fmt"
	"strconv"
	"strings"
)

// A Version represents a version of a model schema
type Version struct {
	Major int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Patch)
}

// Before reports whether v should be ordered before v2
func (v Version) Before(v2 Version) bool {
	return VersionCmp(v, v2) == -1
}

// VersionCmp compares versions a and b and returns -1 if a < b, +1 if a > b and 0 if they are equal
func VersionCmp(a, b Version) int {
	if a.Major < b.Major {
		return -1
	}
	if a.Major > b.Major {
		return 1
	}

	// Major are same, compare patches
	if a.Patch < b.Patch {
		return -1
	}
	if a.Patch > b.Patch {
		return 1
	}
	return 0
}

func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return Version{}, fmt.Errorf("invalid version format: expected major.patch, got %s", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %w", err)
	}
	patch, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %w", err)
	}
	return Version{
		Major: major,
		Patch: patch,
	}, nil
}
