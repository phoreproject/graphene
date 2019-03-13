package utils

// Assert asserts
func Assert(b bool, message string) {
	if !b {
		panic(message)
	}
}
