package communicator

import (
	"context"
	"fmt"
	config "github.com/weldpua2008/supraworker/config"
	utils "github.com/weldpua2008/supraworker/utils"

	"strings"
)

// A Communicator is the interface used to communicate with APIs
// that will eventually return metadata. Communicators
// allow you to get information from remote APi, databases, etc.
//
// Communicators must be safe for concurrency, meaning multiple calls to
// any method may be called at the same time.
type Communicator interface {
	// Configured
	Configured() bool

	// Configure Communicator
	Configure(map[string]interface{}) error
	// Fetch metadata from remote storage
	Fetch(context.Context, map[string]interface{}) ([]map[string]interface{}, error)
}

// GetCommunicator returns Communicator by type.
func GetCommunicator(communicator_type string) (Communicator, error) {
	k := strings.ToUpper(communicator_type)
	if type_struct, ok := Constructors[k]; ok {
		if comm := type_struct.instance(); comm != nil {
			return comm, nil
		} else {
			return nil, fmt.Errorf("%w for %s.\n", ErrNoSuitableCommunicator, communicator_type)
		}
	}

	return nil, fmt.Errorf("%w for %s.\n", ErrNoSuitableCommunicator, communicator_type)
}

// GetSectionCommunicator returns communicator from configuration file.
// By default http communicator will be used.
// Example YAML config for `section` that will return new `RestCommunicator`:
//     section:
//         communicator:
//             type: "HTTP"
func GetSectionCommunicator(section string) (Communicator, error) {
	communicator_type := config.GetStringDefault(fmt.Sprintf("%s.%s.type", section, config.CFG_PREFIX_COMMUNICATOR), "http")
	k := strings.ToUpper(communicator_type)
	if type_struct, ok := Constructors[k]; ok {
		if comm, err := type_struct.constructor(section); err == nil {
			return comm, nil
		} else {
			return nil, err
		}

	}
	return nil, fmt.Errorf("%w for %s.\n", ErrNoSuitableCommunicator, communicator_type)
}

// GetCommunicatorsFromSection returns multiple communicators from configuration file.
// By default http communicator will be used.
// Example YAML config for `section` that will return new `RestCommunicator`:
//     section:
//         communicators:
//             my_communicator:
//                 type: "HTTP"
//             -:
//                 type: "HTTP"
func GetCommunicatorsFromSection(section string) ([]Communicator, error) {
	def := make(map[string]string)

	comms := config.GetMapStringMapStringTemplatedDefault(section, config.CFG_PREFIX_COMMUNICATORS, def)

	res := make([]Communicator, 0)
	for section, comm := range comms {
		if comm == nil {
			continue
		}
		communicator_type := ConstructorsTypeRest
		if comm_type, ok := comm["type"]; ok {
			communicator_type = comm_type
		}
		comm["section"] = section
		if _, ok := comm["param"]; !ok {
			comm["param"] = config.CFG_COMMUNICATOR_PARAMS_KEY
		}

		k := strings.ToUpper(communicator_type)
		if type_struct, ok := Constructors[k]; ok {
			communicator_instance := type_struct.instance()
			if err1 := communicator_instance.Configure(utils.ConvertMapStringToInterface(comm)); err1 != nil {
				log.Tracef("Can't configure %v communicator, got %v", communicator_type, comm)
				return nil, err1
			}
			// log.Tracef("Configured communicator %v with %v",k, comm)
			res = append(res, communicator_instance)
		}
	}
	if len(res) > 0 {
		return res, nil
	}
	return nil, fmt.Errorf("%w in section %s.\n", ErrNoSuitableCommunicator, section)
}
