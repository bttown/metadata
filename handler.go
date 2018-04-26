package metadata

type (
	// Then will be called when metadata got.
	Then func(Request, Torrent)
	// Reject will be called when a error occur.
	Reject func(Request, error)
)
