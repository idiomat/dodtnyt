package basics_test

import (
	"flag"
	"testing"

	"github.com/idiomat/dodtnyt/testing/basics"
)

var doSomethingSpecial = flag.Bool("special", false, "do something special")

func TestSumParallel(t *testing.T) {
	tests := map[string]struct {
		nums     []int
		expected int
	}{
		"positive numbers":             {nums: []int{1, 2, 3}, expected: 6},
		"negative numbers":             {nums: []int{-1, -2, -3}, expected: -6},
		"mix of positive and negative": {nums: []int{-1, 2, -3, 4}, expected: 2},
		"zero values":                  {nums: []int{0, 0, 0}, expected: 0},
		"empty slice":                  {nums: []int{}, expected: 0},
	}

	if *doSomethingSpecial {
		tests["special"] = struct {
			nums     []int
			expected int
		}{nums: []int{1, 2, 3, 4, 5}, expected: 15}
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := basics.Sum(tt.nums)
			if result != tt.expected {
				t.Errorf("Sum(%v) = %d; expected %d", tt.nums, result, tt.expected)
			}
		})
	}
}
