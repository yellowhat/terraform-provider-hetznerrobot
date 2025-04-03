package vswitch

import (
	"slices"
	"sort"
	"testing"
)

func TestDiffServers(t *testing.T) {
	type testCase struct {
		name         string
		oldList      []int
		newList      []int
		wantToAdd    []int
		wantToRemove []int
	}

	testCases := []testCase{
		{
			name:         "No Changes",
			oldList:      []int{1, 2, 3},
			newList:      []int{1, 2, 3},
			wantToAdd:    []int{},
			wantToRemove: []int{},
		},
		{
			name:         "No Changes from unordered",
			oldList:      []int{1, 2, 3},
			newList:      []int{3, 2, 1},
			wantToAdd:    []int{},
			wantToRemove: []int{},
		},
		{
			name:         "Add Servers",
			oldList:      []int{1, 2},
			newList:      []int{1, 2, 3, 4},
			wantToAdd:    []int{3, 4},
			wantToRemove: []int{},
		},
		{
			name:         "Remove Servers",
			oldList:      []int{1, 2, 3, 4},
			newList:      []int{1, 2},
			wantToAdd:    []int{},
			wantToRemove: []int{3, 4},
		},
		{
			name:         "Mixed Add and Remove",
			oldList:      []int{1, 2, 3},
			newList:      []int{2, 3, 4},
			wantToAdd:    []int{4},
			wantToRemove: []int{1},
		},
		{
			name:         "Empty Old List",
			oldList:      []int{},
			newList:      []int{1, 2},
			wantToAdd:    []int{1, 2},
			wantToRemove: []int{},
		},
		{
			name:         "Empty New List",
			oldList:      []int{1, 2},
			newList:      []int{},
			wantToAdd:    []int{},
			wantToRemove: []int{1, 2},
		},
	}

	compareOutput := func(actual, want []int) bool {
		sort.Ints(actual)
		sort.Ints(want)
		return slices.Equal(actual, want)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toAdd, toRemove := diffServers(tc.oldList, tc.newList)
			if !compareOutput(toAdd, tc.wantToAdd) {
				t.Errorf("Test %s: toAdd = %v, want %v", tc.name, toAdd, tc.wantToAdd)
			}
			if !compareOutput(toRemove, tc.wantToRemove) {
				t.Errorf("Test %s: toRemove = %v, want %v", tc.name, toRemove, tc.wantToRemove)
			}
		})
	}
}
