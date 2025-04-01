package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestArgumentsRootCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedIvals []int64
		expectedFvals []float64
		errStr        string
	}{
		{
			name:          "set ival",
			args:          []string{"foo", "10"},
			expectedIvals: []int64{10},
		},
		{
			name:          "set ival fval",
			args:          []string{"foo", "12", "10.1"},
			expectedIvals: []int64{12},
			expectedFvals: []float64{10.1},
		},
		{
			name:          "set ival multu fvals",
			args:          []string{"foo", "13", "10.1", "11.09"},
			expectedIvals: []int64{13},
			expectedFvals: []float64{10.1, 11.09},
		},
		{
			name:          "set fvals beyond max",
			args:          []string{"foo", "13", "10.1", "11.09", "12.1"},
			expectedIvals: []int64{13},
			expectedFvals: []float64{10.1, 11.09},
			errStr:        "No help topic for '12.1'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := buildMinimalTestCommand()
			var ivals []int64
			var fvals []float64
			cmd.Arguments = []Argument{
				&IntArgs{
					Name:        "ia",
					Min:         1,
					Max:         1,
					Destination: &ivals,
				},
				&FloatArgs{
					Name:        "fa",
					Min:         0,
					Max:         2,
					Destination: &fvals,
				},
			}

			err := cmd.Run(buildTestContext(t), test.args)

			r := require.New(t)

			if test.errStr != "" {
				r.ErrorContains(err, test.errStr)
			}
			r.Equal(test.expectedIvals, ivals)
			r.Equal(test.expectedIvals, cmd.IntArgs("ia"))
			r.Equal(test.expectedFvals, fvals)
			if test.expectedFvals != nil {
				r.Equal(test.expectedFvals, cmd.FloatArgs("fa"))
			} else {
				r.Equal([]float64{}, cmd.FloatArgs("fa"))
			}
		})
	}

	/*
	   cmd.Arguments = append(cmd.Arguments,

	   	&StringArgs{
	   		Name: "sa",
	   	},
	   	&UintArgs{
	   		Name: "ua",
	   		Min:  2,
	   		Max:  1, // max is less than min
	   	},

	   )

	   require.NoError(t, cmd.Run(context.Background(), []string{"foo", "10"}))
	*/
}

func TestArgumentsSubcommand(t *testing.T) {
	cmd := buildMinimalTestCommand()
	var ifval int64
	var svals []string
	var tvals []time.Time
	cmd.Commands = []*Command{
		{
			Name: "subcmd",
			Flags: []Flag{
				&IntFlag{
					Name:        "foo",
					Value:       10,
					Destination: &ifval,
				},
			},
			Arguments: []Argument{
				&TimestampArgs{
					Name:        "ta",
					Min:         1,
					Max:         1,
					Destination: &tvals,
					Config: TimestampConfig{
						Layouts: []string{time.RFC3339},
					},
				},
				&StringArgs{
					Name:        "sa",
					Min:         1,
					Max:         3,
					Destination: &svals,
				},
			},
		},
	}

	numUsageErrors := 0
	cmd.Commands[0].OnUsageError = func(ctx context.Context, cmd *Command, err error, isSubcommand bool) error {
		numUsageErrors++
		return err
	}

	require.Error(t, errors.New("sufficient count of arg sa not provided, given 0 expected 1"), cmd.Run(context.Background(), []string{"foo", "subcmd", "2006-01-02T15:04:05Z"}))
	require.Equal(t, 1, numUsageErrors)

	tvals = []time.Time{}
	require.NoError(t, cmd.Run(context.Background(), []string{"foo", "subcmd", "2006-01-02T15:04:05Z", "fubar"}))
	require.Equal(t, []time.Time{time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)}, tvals)
	require.Equal(t, []string{"fubar"}, svals)

	tvals = []time.Time{}
	svals = []string{}
	require.NoError(t, cmd.Run(context.Background(), []string{"foo", "subcmd", "--foo", "100", "2006-01-02T15:04:05Z", "fubar", "some"}))
	require.Equal(t, int64(100), ifval)
	require.Equal(t, []time.Time{time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)}, tvals)
	require.Equal(t, []string{"fubar", "some"}, svals)
}

func TestArgsUsage(t *testing.T) {
	arg := &IntArgs{
		Name: "ia",
		Min:  0,
		Max:  1,
	}
	tests := []struct {
		name     string
		min      int
		max      int
		usage    string
		expected string
	}{
		{
			name:     "optional",
			min:      0,
			max:      1,
			expected: "[ia]",
		},
		{
			name:     "optional",
			min:      0,
			max:      1,
			usage:    "[my optional usage]",
			expected: "[my optional usage]",
		},
		{
			name:     "zero or more",
			min:      0,
			max:      2,
			expected: "[ia ...]",
		},
		{
			name:     "one",
			min:      1,
			max:      1,
			expected: "ia [ia ...]",
		},
		{
			name:     "many",
			min:      2,
			max:      1,
			expected: "ia [ia ...]",
		},
		{
			name:     "many2",
			min:      2,
			max:      0,
			expected: "ia [ia ...]",
		},
		{
			name:     "unlimited",
			min:      2,
			max:      -1,
			expected: "ia [ia ...]",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			arg.Min, arg.Max, arg.UsageText = test.min, test.max, test.usage
			require.Equal(t, test.expected, arg.Usage())
		})
	}
}

func TestSingleOptionalArg(t *testing.T) {
	cmd := buildMinimalTestCommand()
	var s1 []string
	arg := &StringArgs{
		Min:         0,
		Max:         1,
		Destination: &s1,
	}
	cmd.Arguments = []Argument{
		arg,
	}

	require.NoError(t, cmd.Run(context.Background(), []string{"foo"}))
	require.Equal(t, []string{}, s1)

	/*arg.Value = "bar"
	require.NoError(t, cmd.Run(context.Background(), []string{"foo"}))
	require.Equal(t, "bar", s1)*/

	require.NoError(t, cmd.Run(context.Background(), []string{"foo", "zbar"}))
	require.Equal(t, []string{"zbar"}, s1)
}

func TestUnboundedArgs(t *testing.T) {
	arg := &StringArgs{
		Min: 0,
		Max: -1,
	}
	tests := []struct {
		name     string
		args     []string
		values   []string
		expected []string
	}{
		{
			name:     "cmd accepts no args",
			args:     []string{"foo"},
			expected: []string{},
		},
		{
			name:     "cmd uses given args",
			args:     []string{"foo", "bar", "baz"},
			expected: []string{"bar", "baz"},
		},
		{
			name:     "cmd uses default values",
			args:     []string{"foo"},
			values:   []string{"zbar", "zbaz"},
			expected: []string{"zbar", "zbaz"},
		},
		{
			name:     "given args override default values",
			args:     []string{"foo", "bar", "baz"},
			values:   []string{"zbar", "zbaz"},
			expected: []string{"bar", "baz"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := buildMinimalTestCommand()
			cmd.Arguments = []Argument{arg}
			arg.Destination = &test.values
			require.NoError(t, cmd.Run(context.Background(), test.args))
			require.Equal(t, test.expected, *arg.Destination)
		})
	}
}
