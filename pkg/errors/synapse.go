package errors

var (
	// ErrSynapseNotFound is returned when there is no available synpase
	ErrSynapseNotFound = New("No Synapse avaialbale for given resource constraint")

	// ErrSynapseMessageExceedAllotedTime is returned when message is in queue for too long
	ErrSynapseMessageExceedAllotedTime = New("Message exceed time duration")
	// ErrSynapseDuplicateConnection  will be thrown when synapse try to connect to neuron and there is already one connection open
	ErrSynapseDuplicateConnection = New("Synapse already has an open connection")
	// ErrSynapseAuthFailed should be thrown when authentication failed
	ErrSynapseAuthFailed = New("Synapse authentication failed")
	// ErrSynapseMinRequirement will be thrown if synapse does not meet minimum requirement
	ErrSynapseMinRequirement = New("Synapse does not meet min requirement")
)
