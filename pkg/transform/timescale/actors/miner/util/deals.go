package util

import "github.com/filecoin-project/go-state-types/abi"

func CompareDealIDs(cur, pre []abi.DealID) []abi.DealID {
	var diff []abi.DealID

	// Loop two times, first to find cur dealIDs not in pre,
	// second loop to find pre dealIDs not in cur
	for i := 0; i < 2; i++ {
		for _, s1 := range cur {
			found := false
			for _, s2 := range pre {
				if s1 == s2 {
					found = true
					break
				}
			}
			// DealID not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			cur, pre = pre, cur
		}
	}

	return diff
}
