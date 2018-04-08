package metadata

type (
	// Handler ...
	Handler func(*Request, *Metadata)
	// ErrHandler ...
	ErrHandler func(*Request, error)
)
