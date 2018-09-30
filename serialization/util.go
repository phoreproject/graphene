package serialization

func AppendAll(elems ...[]byte) []byte {
	out := []byte{}
	for _, e := range elems {
		out = append(out, e...)
	}
	return out
}
