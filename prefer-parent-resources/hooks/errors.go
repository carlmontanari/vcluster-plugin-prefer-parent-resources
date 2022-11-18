package hooks

import "errors"

var (
	// ErrWrongResourceType is an error that is returned when the mutate hook encounters an
	// unexpected/undesired type -- generally this should not be encountered.
	ErrWrongResourceType = errors.New("errWrongResourceType")
	// ErrCantGetResource is an error returned when unable to find a given resource in either the
	// parent/physical cluster or the vcluster.
	ErrCantGetResource = errors.New("errCantGetResource")
)
