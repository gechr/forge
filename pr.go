package forge

import (
	"fmt"
	"strconv"
	"strings"
)

// pullRequestRange is an inclusive [lo, hi] pull request number range; a
// single number is a range where lo == hi.
type pullRequestRange struct {
	lo int
	hi int
}

// ExpandPRList expands a PR list like "1,2,5-7" into []int{1, 2, 5, 6, 7}.
// Ranges are inclusive on both ends; duplicates are dropped; negative or
// zero numbers are rejected.
func ExpandPRList(spec string) ([]int, error) {
	var nums []int
	seen := make(map[int]struct{})
	for segment := range strings.SplitSeq(spec, ",") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, fmt.Errorf("empty PR number in %q", spec)
		}
		r, ok := parsePullRequestRange(segment)
		if !ok {
			return nil, fmt.Errorf("invalid PR reference %q", segment)
		}
		for n := r.lo; n <= r.hi; n++ {
			if _, dup := seen[n]; dup {
				continue
			}
			seen[n] = struct{}{}
			nums = append(nums, n)
		}
	}
	return nums, nil
}

func parsePullRequestRange(segment string) (pullRequestRange, bool) {
	if before, after, hasDash := strings.Cut(segment, "-"); hasDash {
		lo, errLo := strconv.Atoi(before)
		hi, errHi := strconv.Atoi(after)
		if errLo != nil || errHi != nil || lo <= 0 || hi < lo {
			return pullRequestRange{}, false
		}
		return pullRequestRange{lo: lo, hi: hi}, true
	}
	n, err := strconv.Atoi(segment)
	if err != nil || n <= 0 {
		return pullRequestRange{}, false
	}
	return pullRequestRange{lo: n, hi: n}, true
}
