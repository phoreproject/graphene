package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRanges parses a list of ranges into a list of included indices.
func ParseRanges(validatorsStrings []string) ([]uint32, error) {
	var validatorIndices []uint32
	validatorIndicesMap := map[int]struct{}{}
	for _, s := range validatorsStrings {
		if !strings.ContainsRune(s, '-') {
			i, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("invalid validators parameter")
			}
			validatorIndicesMap[i] = struct{}{}
			validatorIndices = append(validatorIndices, uint32(i))
		} else {
			parts := strings.SplitN(s, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid validators parameter")
			}
			first, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid validators parameter")
			}
			second, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid validators parameter")
			}
			for i := first; i <= second; i++ {
				validatorIndices = append(validatorIndices, uint32(i))
				validatorIndicesMap[i] = struct{}{}
			}
		}
	}

	return validatorIndices, nil
}