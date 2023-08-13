package pprof_test

import (
	"bytes"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/macabu/cpgo/internal/pprof"
)

func TestMergeProfiles(t *testing.T) {
	profileValid := &profile.Profile{
		TimeNanos:     10000,
		PeriodType:    &profile.ValueType{Type: "cpu", Unit: "milliseconds"},
		Period:        1,
		DurationNanos: 10e9,
		SampleType: []*profile.ValueType{
			{Type: "samples", Unit: "count"},
			{Type: "cpu", Unit: "milliseconds"},
		},
		Sample: []*profile.Sample{},
	}

	var w bytes.Buffer

	err := pprof.MergeProfiles(&w, []*profile.Profile{profileValid, profileValid})
	require.NoError(t, err)
	require.NotNil(t, w.Bytes())
}
