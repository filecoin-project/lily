package model

import (
	"testing"
)

func TestVersionBefore(t *testing.T) {
	positiveTestCases := []struct {
		a, b Version
	}{

		{
			a: Version{Major: 0, Patch: 0},
			b: Version{Major: 1, Patch: 0},
		},
		{
			a: Version{Major: 1, Patch: 0},
			b: Version{Major: 1, Patch: 1},
		},
		{
			a: Version{Major: 1, Patch: 1},
			b: Version{Major: 2, Patch: 0},
		},
	}

	negativeTestCases := []struct {
		a, b Version
	}{

		{
			a: Version{Major: 1, Patch: 0},
			b: Version{Major: 0, Patch: 0},
		},
		{
			a: Version{Major: 1, Patch: 0},
			b: Version{Major: 1, Patch: 0},
		},
		{
			a: Version{Major: 1, Patch: 1},
			b: Version{Major: 1, Patch: 0},
		},
	}

	for _, tc := range positiveTestCases {
		if !tc.a.Before(tc.b) {
			t.Errorf("got %s before %s = false, wanted true", tc.a, tc.b)
		}
	}

	for _, tc := range negativeTestCases {
		if tc.a.Before(tc.b) {
			t.Errorf("got %s before %s = true, wanted false", tc.a, tc.b)
		}
	}
}

func TestVersionCmp(t *testing.T) {
	testCases := []struct {
		a, b Version
		cmp  int
	}{

		{
			a:   Version{Major: 0, Patch: 0},
			b:   Version{Major: 1, Patch: 0},
			cmp: -1,
		},
		{
			a:   Version{Major: 1, Patch: 0},
			b:   Version{Major: 1, Patch: 1},
			cmp: -1,
		},
		{
			a:   Version{Major: 1, Patch: 1},
			b:   Version{Major: 2, Patch: 0},
			cmp: -1,
		},
		{
			a:   Version{Major: 1, Patch: 0},
			b:   Version{Major: 0, Patch: 0},
			cmp: 1,
		},
		{
			a:   Version{Major: 1, Patch: 0},
			b:   Version{Major: 1, Patch: 0},
			cmp: 0,
		},
		{
			a:   Version{Major: 1, Patch: 1},
			b:   Version{Major: 1, Patch: 0},
			cmp: 1,
		},
		{
			a:   Version{Major: 1, Patch: 1},
			b:   Version{Major: 1, Patch: 1},
			cmp: 0,
		},
	}

	for _, tc := range testCases {
		cmp := VersionCmp(tc.a, tc.b)
		if cmp != tc.cmp {
			t.Errorf("got VersionCmp(%s,%s)=%d, wanted %d", tc.a, tc.b, cmp, tc.cmp)
		}
	}
}

func TestParseVersion(t *testing.T) {
	testCases := []struct {
		input   string
		version Version
		err     bool
	}{
		{
			input:   "0.1",
			version: Version{Major: 0, Patch: 1},
		},
		{
			input:   "1.0",
			version: Version{Major: 1, Patch: 0},
		},
		{
			input:   "1.1221",
			version: Version{Major: 1, Patch: 1221},
		},
		{
			input: "1.0.1",
			err:   true,
		},
		{
			input: "v1.0",
			err:   true,
		},
		{
			input: "1",
			err:   true,
		},
		{
			input: "1 0",
			err:   true,
		},
	}

	for _, tc := range testCases {
		v, err := ParseVersion(tc.input)
		switch {
		case tc.err && err == nil:
			t.Errorf("ParseVersion(%q) gave no error, wanted one", tc.input)
		case !tc.err && err != nil:
			t.Errorf("ParseVersion(%q) returned unexepected error %v", tc.input, err)
		}
		if v != tc.version {
			t.Errorf("ParseVersion(%q)=%+v, wanted %+v", tc.input, v, tc.version)
		}
	}
}
