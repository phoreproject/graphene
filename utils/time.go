package utils

import "time"

// GetCurrentMilliseconds gets current milliseconds
func GetCurrentMilliseconds() uint64 {
	return uint64(time.Now().UnixNano()) / uint64(1000000)
}
