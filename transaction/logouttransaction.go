package transaction

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From      uint32
	Signature []byte
}
