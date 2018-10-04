package serialization

// AppendAll appends many []byte's together.
func AppendAll(elems ...[]byte) []byte {
	out := []byte{}
	for _, e := range elems {
		out = append(out, e...)
	}
	return out
}
