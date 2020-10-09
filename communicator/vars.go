package communicator

import (
	"errors"
	"github.com/sirupsen/logrus"
)

var (
	// Constructors is a map of all Communicator types with their specs.
	Constructors = map[string]TypeSpec{}

	ErrNoSuitableCommunicator  = errors.New("No suitable communicator found")
	ErrFailedSendRequest       = errors.New("Failed to send request")
	ErrFailedReadResponseBody  = errors.New("Failed to read response body")
	ErrFailedUnmarshalResponse = errors.New("Cannot unmarshal response")
	ErrFailedMarshalRequest    = errors.New("Cannot marshal request")
	ErrFailedResponseCode      = errors.New("Failed request's response code")
	ErrNotAllowedResponseCode  = errors.New("Not allowed response code")

	// internal
	log = logrus.WithFields(logrus.Fields{"package": "communicator"})
)

// String constants representing each communicator type.
const (
	// ConstructorsTypeRest represents HTTP communicator
	ConstructorsTypeRest       = "HTTP"
	CTX_ALLOWED_RESPONSE_CODES = "allowed_response_codes"
	CFG_PREFIX_COMMUNICATORS   = ""
)
