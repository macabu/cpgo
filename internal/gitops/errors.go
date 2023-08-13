package gitops

import "errors"

var ErrPGOFileNotFound = errors.New("could not find existing PGO file")
