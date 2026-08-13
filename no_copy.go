package ringbuffer

// noCopy may be embedded into structs that must not be copied after first use.
// The Lock and Unlock methods allow go vet's copylocks analyzer to detect copies.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
