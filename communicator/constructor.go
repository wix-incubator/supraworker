package communicator

// TypeSpec is a constructor and a usage description for each Communicator type.
type TypeSpec struct {
	constructor func(string) (Communicator, error)
	instance    func() Communicator
	Summary     string
	Description string
	Beta        bool
	Deprecated  bool
}
