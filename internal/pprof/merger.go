package pprof

import (
	"fmt"
	"io"

	"github.com/google/pprof/profile"
)

// MergeProfiles writes the merged uncompressed profile into `w`.
func MergeProfiles(w io.Writer, profiles []*profile.Profile) error {
	mergedProfile, err := profile.Merge(profiles)
	if err != nil {
		return fmt.Errorf("profile.Merge: %w", err)
	}

	if err := mergedProfile.WriteUncompressed(w); err != nil {
		return fmt.Errorf("mergedProfile.WriteUncompressed: %w", err)
	}

	return nil
}
