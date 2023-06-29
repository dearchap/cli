package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var boolFlagTests = []struct {
	name     string
	expected string
}{
	{"help", "--help\t(default: false)"},
	{"h", "-h\t(default: false)"},
}

func resetEnv(env []string) {
	for _, e := range env {
		fields := strings.SplitN(e, "=", 2)
		os.Setenv(fields[0], fields[1])
	}
}

func TestBoolFlagHelpOutput(t *testing.T) {
	for _, test := range boolFlagTests {
		fl := &BoolFlag{Name: test.name}
		output := fl.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestBoolFlagApply_SetsAllNames(t *testing.T) {
	v := false
	fl := BoolFlag{Name: "wat", Aliases: []string{"W", "huh"}, Destination: &v}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--wat", "-W", "--huh"})
	expect(t, err, nil)
	expect(t, v, true)
}

func TestBoolFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Bool("trueflag", true, "doc")
	set.Bool("falseflag", false, "doc")
	cmd := &Command{flagSet: set}
	tf := &BoolFlag{Name: "trueflag"}
	ff := &BoolFlag{Name: "falseflag"}

	r := require.New(t)
	r.True(tf.Get(cmd))
	r.False(ff.Get(cmd))
}

func TestBoolFlagApply_SetsCount(t *testing.T) {
	v := false
	count := 0
	fl := BoolFlag{Name: "wat", Aliases: []string{"W", "huh"}, Destination: &v, Config: BoolConfig{Count: &count}}
	set := flag.NewFlagSet("test", 0)
	err := fl.Apply(set)
	expect(t, err, nil)

	err = set.Parse([]string{"--wat", "-W", "--huh"})
	expect(t, err, nil)
	expect(t, v, true)
	expect(t, count, 3)
}

func TestBoolFlagCountFromCommand(t *testing.T) {
	boolCountTests := []struct {
		input         []string
		expectedVal   bool
		expectedCount int
	}{
		{
			input:         []string{"-tf", "-w", "-huh"},
			expectedVal:   true,
			expectedCount: 3,
		},
		{
			input:         []string{},
			expectedVal:   false,
			expectedCount: 0,
		},
	}

	for _, bct := range boolCountTests {
		set := flag.NewFlagSet("test", 0)
		cmd := &Command{flagSet: set}
		tf := &BoolFlag{Name: "tf", Aliases: []string{"w", "huh"}}
		r := require.New(t)

		r.NoError(tf.Apply(set))
		r.NoError(set.Parse(bct.input))

		r.Equal(bct.expectedVal, tf.Get(cmd))
		r.Equal(bct.expectedCount, cmd.Count("tf"))
	}
}

func TestFlagsFromEnv(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		output      any
		fl          Flag
		errContains string
	}{
		{
			name:   "BoolFlag valid true",
			input:  "1",
			output: true,
			fl:     &BoolFlag{Name: "debug", Sources: EnvVars("DEBUG")},
		},
		{
			name:   "BoolFlag valid false",
			input:  "false",
			output: false,
			fl:     &BoolFlag{Name: "debug", Sources: EnvVars("DEBUG")},
		},
		{
			name:   "BoolFlag invalid",
			input:  "foobar",
			output: true,
			fl:     &BoolFlag{Name: "debug", Sources: EnvVars("DEBUG")},
			errContains: `could not parse "foobar" as bool value from environment variable ` +
				`"DEBUG" for flag debug:`,
		},

		{
			name:   "DurationFlag valid",
			input:  "1s",
			output: 1 * time.Second,
			fl:     &DurationFlag{Name: "time", Sources: EnvVars("TIME")},
		},
		{
			name:   "DurationFlag invalid",
			input:  "foobar",
			output: false,
			fl:     &DurationFlag{Name: "time", Sources: EnvVars("TIME")},
			errContains: `could not parse "foobar" as time.Duration value from environment ` +
				`variable "TIME" for flag time:`,
		},

		{
			name:   "Float64Flag valid",
			input:  "1.2",
			output: 1.2,
			fl:     &FloatFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "Float64Flag valid from int",
			input:  "1",
			output: 1.0,
			fl:     &FloatFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "Float64Flag invalid",
			input:  "foobar",
			output: 0,
			fl:     &FloatFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "foobar" as float64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},

		{
			name:   "IntFlag valid",
			input:  "1",
			output: int64(1),
			fl:     &IntFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "IntFlag invalid from float",
			input:  "1.2",
			output: 0,
			fl:     &IntFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "1.2" as int64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
		{
			name:   "IntFlag invalid",
			input:  "foobar",
			output: 0,
			fl:     &IntFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "foobar" as int64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
		{
			name:   "IntFlag valid from hex",
			input:  "deadBEEF",
			output: int64(3735928559),
			fl:     &IntFlag{Name: "seconds", Sources: EnvVars("SECONDS"), Config: IntegerConfig{Base: 16}},
		},
		{
			name:   "IntFlag invalid from octal",
			input:  "08",
			output: 0,
			fl:     &IntFlag{Name: "seconds", Sources: EnvVars("SECONDS"), Config: IntegerConfig{Base: 0}},
			errContains: `could not parse "08" as int64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},

		{
			name:   "Float64SliceFlag valid",
			input:  "1.0,2",
			output: []float64{1, 2},
			fl:     &FloatSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "Float64SliceFlag invalid",
			input:  "foobar",
			output: []float64{},
			fl:     &FloatSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "foobar" as []float64 value from environment ` +
				`variable "SECONDS" for flag seconds:`,
		},

		{
			name:   "IntSliceFlag valid",
			input:  "1,2",
			output: []int64{1, 2},
			fl:     &IntSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "IntSliceFlag invalid from float",
			input:  "1.2,2",
			output: []int64{},
			fl:     &IntSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "1.2,2" as []int64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
		{
			name:   "IntSliceFlag invalid",
			input:  "foobar",
			output: []int64{},
			fl:     &IntSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "foobar" as []int64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},

		{
			name:   "UintSliceFlag valid",
			input:  "1,2",
			output: []uint64{1, 2},
			fl:     &UintSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "UintSliceFlag invalid with float",
			input:  "1.2,2",
			output: []uint64{},
			fl:     &UintSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "1.2,2" as []uint64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
		{
			name:   "UintSliceFlag invalid",
			input:  "foobar",
			output: []uint64{},
			fl:     &UintSliceFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "foobar" as []uint64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},

		{
			name:   "StringFlag valid",
			input:  "foo",
			output: "foo",
			fl:     &StringFlag{Name: "name", Sources: EnvVars("NAME")},
		},
		{
			name:   "StringFlag valid with TrimSpace",
			input:  " foo",
			output: "foo",
			fl:     &StringFlag{Name: "names", Sources: EnvVars("NAMES"), Config: StringConfig{TrimSpace: true}},
		},

		{
			name:   "StringSliceFlag valid",
			input:  "foo,bar",
			output: []string{"foo", "bar"},
			fl:     &StringSliceFlag{Name: "names", Sources: EnvVars("NAMES")},
		},
		{
			name:   "StringSliceFlag valid with TrimSpace",
			input:  "foo , bar ",
			output: []string{"foo", "bar"},
			fl:     &StringSliceFlag{Name: "names", Sources: EnvVars("NAMES"), Config: StringConfig{TrimSpace: true}},
		},

		{
			name:   "StringMapFlag valid",
			input:  "foo=bar,empty=",
			output: map[string]string{"foo": "bar", "empty": ""},
			fl:     &StringMapFlag{Name: "names", Sources: EnvVars("NAMES")},
		},
		{
			name:   "StringMapFlag valid with TrimSpace",
			input:  "foo= bar ",
			output: map[string]string{"foo": "bar"},
			fl:     &StringMapFlag{Name: "names", Sources: EnvVars("NAMES"), Config: StringConfig{TrimSpace: true}},
		},

		{
			name:   "UintFlag valid",
			input:  "1",
			output: uint64(1),
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
		},
		{
			name:   "UintFlag valid leading zero",
			input:  "08",
			output: uint64(8),
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS"), Config: IntegerConfig{Base: 10}},
		},
		{
			name:   "UintFlag valid from octal",
			input:  "755",
			output: uint64(493),
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS"), Config: IntegerConfig{Base: 8}},
		},
		{
			name:   "UintFlag valid from hex",
			input:  "deadBEEF",
			output: uint64(3735928559),
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS"), Config: IntegerConfig{Base: 16}},
		},
		{
			name:   "UintFlag invalid octal",
			input:  "08",
			output: 0,
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS"), Config: IntegerConfig{Base: 0}},
			errContains: `could not parse "08" as uint64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
		{
			name:   "UintFlag invalid float",
			input:  "1.2",
			output: 0,
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "1.2" as uint64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
		{
			name:   "UintFlag invalid",
			input:  "foobar",
			output: 0,
			fl:     &UintFlag{Name: "seconds", Sources: EnvVars("SECONDS")},
			errContains: `could not parse "foobar" as uint64 value from environment variable ` +
				`"SECONDS" for flag seconds:`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)

			r.Implements((*DocGenerationFlag)(nil), tc.fl)
			f := tc.fl.(DocGenerationFlag)

			envVarSlice := f.GetEnvVars()
			t.Setenv(envVarSlice[0], tc.input)

			cmd := &Command{
				Flags: []Flag{tc.fl},
				Action: func(_ context.Context, cmd *Command) error {
					r.Equal(cmd.Value(tc.fl.Names()[0]), tc.output)
					r.True(tc.fl.IsSet())
					r.Equal(cmd.FlagNames(), tc.fl.Names())

					return nil
				},
			}

			err := cmd.Run(buildTestContext(t), []string{"run"})

			if tc.errContains != "" {
				r.NotNil(err)
				r.ErrorContains(err, tc.errContains)

				return
			}

			r.NoError(err)
		})
	}
}

type nodocFlag struct {
	Flag

	Name string
}

func TestFlagStringifying(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fl       Flag
		expected string
	}{
		{
			name:     "bool-flag",
			fl:       &BoolFlag{Name: "vividly"},
			expected: "--vividly\t(default: false)",
		},
		{
			name:     "bool-flag-with-default-text",
			fl:       &BoolFlag{Name: "wildly", DefaultText: "scrambled"},
			expected: "--wildly\t(default: scrambled)",
		},
		{
			name:     "duration-flag",
			fl:       &DurationFlag{Name: "scream-for"},
			expected: "--scream-for value\t(default: 0s)",
		},
		{
			name:     "duration-flag-with-default-text",
			fl:       &DurationFlag{Name: "feels-about", DefaultText: "whimsically"},
			expected: "--feels-about value\t(default: whimsically)",
		},
		{
			name:     "float64-flag",
			fl:       &FloatFlag{Name: "arduous"},
			expected: "--arduous value\t(default: 0)",
		},
		{
			name:     "float64-flag-with-default-text",
			fl:       &FloatFlag{Name: "filibuster", DefaultText: "42"},
			expected: "--filibuster value\t(default: 42)",
		},
		{
			name:     "float64-slice-flag",
			fl:       &FloatSliceFlag{Name: "pizzas"},
			expected: "--pizzas value [ --pizzas value ]\t",
		},
		{
			name:     "float64-slice-flag-with-default-text",
			fl:       &FloatSliceFlag{Name: "pepperonis", DefaultText: "shaved"},
			expected: "--pepperonis value [ --pepperonis value ]\t(default: shaved)",
		},
		{
			name:     "int-flag",
			fl:       &IntFlag{Name: "grubs"},
			expected: "--grubs value\t(default: 0)",
		},
		{
			name:     "int-flag-with-default-text",
			fl:       &IntFlag{Name: "poisons", DefaultText: "11ty"},
			expected: "--poisons value\t(default: 11ty)",
		},
		{
			name:     "int-slice-flag",
			fl:       &IntSliceFlag{Name: "pencils"},
			expected: "--pencils value [ --pencils value ]\t",
		},
		{
			name:     "int-slice-flag-with-default-text",
			fl:       &IntFlag{Name: "pens", DefaultText: "-19"},
			expected: "--pens value\t(default: -19)",
		},
		{
			name:     "uint-slice-flag",
			fl:       &UintSliceFlag{Name: "pencils"},
			expected: "--pencils value [ --pencils value ]\t",
		},
		{
			name:     "uint-slice-flag-with-default-text",
			fl:       &UintFlag{Name: "pens", DefaultText: "29"},
			expected: "--pens value\t(default: 29)",
		},
		{
			name:     "int64-flag",
			fl:       &IntFlag{Name: "flume"},
			expected: "--flume value\t(default: 0)",
		},
		{
			name:     "int64-flag-with-default-text",
			fl:       &IntFlag{Name: "shattering", DefaultText: "22"},
			expected: "--shattering value\t(default: 22)",
		},
		{
			name:     "uint64-slice-flag",
			fl:       &UintSliceFlag{Name: "drawers"},
			expected: "--drawers value [ --drawers value ]\t",
		},
		{
			name:     "uint64-slice-flag-with-default-text",
			fl:       &UintSliceFlag{Name: "handles", DefaultText: "-2"},
			expected: "--handles value [ --handles value ]\t(default: -2)",
		},
		{
			name:     "string-flag",
			fl:       &StringFlag{Name: "arf-sound"},
			expected: "--arf-sound value\t",
		},
		{
			name:     "string-flag-with-default-text",
			fl:       &StringFlag{Name: "woof-sound", DefaultText: "urp"},
			expected: "--woof-sound value\t(default: urp)",
		},
		{
			name:     "string-slice-flag",
			fl:       &StringSliceFlag{Name: "meow-sounds"},
			expected: "--meow-sounds value [ --meow-sounds value ]\t",
		},
		{
			name:     "string-slice-flag-with-default-text",
			fl:       &StringSliceFlag{Name: "moo-sounds", DefaultText: "awoo"},
			expected: "--moo-sounds value [ --moo-sounds value ]\t(default: awoo)",
		},
		{
			name:     "timestamp-flag",
			fl:       &TimestampFlag{Name: "eating"},
			expected: "--eating value\t",
		},
		{
			name:     "timestamp-flag-with-default-text",
			fl:       &TimestampFlag{Name: "sleeping", DefaultText: "earlier"},
			expected: "--sleeping value\t(default: earlier)",
		},
		{
			name:     "uint-flag",
			fl:       &UintFlag{Name: "jars"},
			expected: "--jars value\t(default: 0)",
		},
		{
			name:     "uint-flag-with-default-text",
			fl:       &UintFlag{Name: "bottles", DefaultText: "99"},
			expected: "--bottles value\t(default: 99)",
		},
		{
			name:     "uint64-flag",
			fl:       &UintFlag{Name: "cans"},
			expected: "--cans value\t(default: 0)",
		},
		{
			name:     "uint64-flag-with-default-text",
			fl:       &UintFlag{Name: "tubes", DefaultText: "13"},
			expected: "--tubes value\t(default: 13)",
		},
		{
			name:     "nodoc-flag",
			fl:       &nodocFlag{Name: "scarecrow"},
			expected: "",
		},
	} {
		t.Run(tc.name, func(ct *testing.T) {
			s := stringifyFlag(tc.fl)
			if s != tc.expected {
				ct.Errorf("stringified flag %q does not match expected %q", s, tc.expected)
			}
		})
	}
}

var stringFlagTests = []struct {
	name     string
	aliases  []string
	usage    string
	value    string
	expected string
}{
	{"foo", nil, "", "", "--foo value\t"},
	{"f", nil, "", "", "-f value\t"},
	{"f", nil, "The total `foo` desired", "all", "-f foo\tThe total foo desired (default: \"all\")"},
	{"test", nil, "", "Something", "--test value\t(default: \"Something\")"},
	{"config", []string{"c"}, "Load configuration from `FILE`", "", "--config FILE, -c FILE\tLoad configuration from FILE"},
	{"config", []string{"c"}, "Load configuration from `CONFIG`", "config.json", "--config CONFIG, -c CONFIG\tLoad configuration from CONFIG (default: \"config.json\")"},
}

func TestStringFlagHelpOutput(t *testing.T) {
	for _, test := range stringFlagTests {
		fl := &StringFlag{Name: test.name, Aliases: test.aliases, Usage: test.usage, Value: test.value}
		// create a tmp flagset
		tfs := flag.NewFlagSet("test", 0)
		if err := fl.Apply(tfs); err != nil {
			t.Error(err)
			return
		}
		output := fl.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestStringFlagDefaultText(t *testing.T) {
	fl := &StringFlag{Name: "foo", Aliases: nil, Usage: "amount of `foo` requested", Value: "none", DefaultText: "all of it"}
	expected := "--foo foo\tamount of foo requested (default: all of it)"
	output := fl.String()

	if output != expected {
		t.Errorf("%q does not match %q", output, expected)
	}
}

func TestStringFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_FOO", "derp")

	for _, test := range stringFlagTests {
		fl := &StringFlag{Name: test.name, Aliases: test.aliases, Value: test.value, Sources: EnvVars("APP_FOO")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_FOO"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%s does not end with"+expectedSuffix, output)
		}
	}
}

var _ = []struct {
	name     string
	aliases  []string
	usage    string
	value    string
	prefixer FlagNamePrefixFunc
	expected string
}{
	{name: "foo", usage: "", value: "", prefixer: func(a []string, b string) string {
		return fmt.Sprintf("name: %s, ph: %s", a, b)
	}, expected: "name: foo, ph: value\t"},
	{name: "f", usage: "", value: "", prefixer: func(a []string, b string) string {
		return fmt.Sprintf("name: %s, ph: %s", a, b)
	}, expected: "name: f, ph: value\t"},
	{name: "f", usage: "The total `foo` desired", value: "all", prefixer: func(a []string, b string) string {
		return fmt.Sprintf("name: %s, ph: %s", a, b)
	}, expected: "name: f, ph: foo\tThe total foo desired (default: \"all\")"},
	{name: "test", usage: "", value: "Something", prefixer: func(a []string, b string) string {
		return fmt.Sprintf("name: %s, ph: %s", a, b)
	}, expected: "name: test, ph: value\t(default: \"Something\")"},
	{name: "config", aliases: []string{"c"}, usage: "Load configuration from `FILE`", value: "", prefixer: func(a []string, b string) string {
		return fmt.Sprintf("name: %s, ph: %s", a, b)
	}, expected: "name: config,c, ph: FILE\tLoad configuration from FILE"},
	{name: "config", aliases: []string{"c"}, usage: "Load configuration from `CONFIG`", value: "config.json", prefixer: func(a []string, b string) string {
		return fmt.Sprintf("name: %s, ph: %s", a, b)
	}, expected: "name: config,c, ph: CONFIG\tLoad configuration from CONFIG (default: \"config.json\")"},
}

func TestStringFlagApply_SetsAllNames(t *testing.T) {
	v := "mmm"
	fl := StringFlag{Name: "hay", Aliases: []string{"H", "hayyy"}, Destination: &v}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--hay", "u", "-H", "yuu", "--hayyy", "YUUUU"})
	expect(t, err, nil)
	expect(t, v, "YUUUU")
}

func TestStringFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.String("myflag", "foobar", "doc")
	cmd := &Command{flagSet: set}
	f := &StringFlag{Name: "myflag"}
	require.Equal(t, "foobar", f.Get(cmd))
}

var _ = []struct {
	name     string
	env      string
	hinter   FlagEnvHintFunc
	expected string
}{
	{"foo", "", func(a []string, b string) string {
		return fmt.Sprintf("env: %s, str: %s", a, b)
	}, "env: , str: --foo value\t"},
	{"f", "", func(a []string, b string) string {
		return fmt.Sprintf("env: %s, str: %s", a, b)
	}, "env: , str: -f value\t"},
	{"foo", "ENV_VAR", func(a []string, b string) string {
		return fmt.Sprintf("env: %s, str: %s", a, b)
	}, "env: ENV_VAR, str: --foo value\t"},
	{"f", "ENV_VAR", func(a []string, b string) string {
		return fmt.Sprintf("env: %s, str: %s", a, b)
	}, "env: ENV_VAR, str: -f value\t"},
}

//func TestFlagEnvHinter(t *testing.T) {
//	defer func() {
//		FlagEnvHinter = withEnvHint
//	}()
//
//	for _, test := range envHintFlagTests {
//		FlagEnvHinter = test.hinter
//		fl := StringFlag{Name: test.name, Sources: ValueSources{test.env}}
//		output := fl.String()
//		if output != test.expected {
//			t.Errorf("%q does not match %q", output, test.expected)
//		}
//	}
//}

var stringSliceFlagTests = []struct {
	name     string
	aliases  []string
	value    []string
	expected string
}{
	{"foo", nil, []string{}, "--foo value [ --foo value ]\t"},
	{"f", nil, []string{}, "-f value [ -f value ]\t"},
	{"f", nil, []string{"Lipstick"}, "-f value [ -f value ]\t(default: \"Lipstick\")"},
	{"test", nil, []string{"Something"}, "--test value [ --test value ]\t(default: \"Something\")"},
	{"dee", []string{"d"}, []string{"Inka", "Dinka", "dooo"}, "--dee value, -d value [ --dee value, -d value ]\t(default: \"Inka\", \"Dinka\", \"dooo\")"},
}

func TestStringSliceFlagHelpOutput(t *testing.T) {
	for _, test := range stringSliceFlagTests {
		f := &StringSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value}
		output := f.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestStringSliceFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_QWWX", "11,4")

	for _, test := range stringSliceFlagTests {
		fl := &StringSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value, Sources: EnvVars("APP_QWWX")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_QWWX"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%q does not end with"+expectedSuffix, output)
		}
	}
}

func TestStringSliceFlagApply_SetsAllNames(t *testing.T) {
	fl := StringSliceFlag{Name: "goat", Aliases: []string{"G", "gooots"}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--goat", "aaa", "-G", "bbb", "--gooots", "eeeee"})
	expect(t, err, nil)
}

func TestStringSliceFlagApply_UsesEnvValues_noDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "vincent van goat,scape goat")
	fl := StringSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT")}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get(), []string{"vincent van goat", "scape goat"})
}

func TestStringSliceFlagApply_UsesEnvValues_withDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "vincent van goat,scape goat")
	val := []string{`some default`, `values here`}
	fl := StringSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)
	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get(), []string{"vincent van goat", "scape goat"})
}

func TestStringSliceFlagApply_DefaultValueWithDestination(t *testing.T) {
	defValue := []string{"UA", "US"}
	dest := []string{"CA"}

	fl := StringSliceFlag{Name: "country", Value: defValue, Destination: &dest}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{})
	expect(t, err, nil)
	expect(t, defValue, dest)
}

func TestStringSliceFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Var(NewStringSlice("a", "b", "c"), "myflag", "doc")
	cmd := &Command{flagSet: set}
	f := &StringSliceFlag{Name: "myflag"}
	require.Equal(t, []string{"a", "b", "c"}, f.Get(cmd))
}

var intFlagTests = []struct {
	name     string
	expected string
}{
	{"hats", "--hats value\t(default: 9)"},
	{"H", "-H value\t(default: 9)"},
}

func TestIntFlagHelpOutput(t *testing.T) {
	for _, test := range intFlagTests {
		fl := &IntFlag{Name: test.name, Value: 9}

		// create a temporary flag set to apply
		tfs := flag.NewFlagSet("test", 0)
		if err := fl.Apply(tfs); err != nil {
			t.Error(err)
			return
		}

		output := fl.String()

		if output != test.expected {
			t.Errorf("%s does not match %s", output, test.expected)
		}
	}
}

func TestIntFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_BAR", "2")

	for _, test := range intFlagTests {
		fl := &IntFlag{Name: test.name, Sources: EnvVars("APP_BAR")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_BAR"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%s does not end with"+expectedSuffix, output)
		}
	}
}

func TestIntFlagApply_SetsAllNames(t *testing.T) {
	v := int64(3)
	fl := IntFlag{Name: "banana", Aliases: []string{"B", "banannanana"}, Destination: &v}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))

	r.NoError(set.Parse([]string{"--banana", "1", "-B", "2", "--banannanana", "5"}))
	r.Equal(int64(5), v)
}

func TestIntFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Int64("myflag", int64(42), "doc")
	cmd := &Command{flagSet: set}
	fl := &IntFlag{Name: "myflag"}
	require.Equal(t, int64(42), fl.Get(cmd))
}

var uintFlagTests = []struct {
	name     string
	expected string
}{
	{"nerfs", "--nerfs value\t(default: 41)"},
	{"N", "-N value\t(default: 41)"},
}

func TestUintFlagHelpOutput(t *testing.T) {
	for _, test := range uintFlagTests {
		fl := &UintFlag{Name: test.name, Value: 41}

		// create a temporary flag set to apply
		tfs := flag.NewFlagSet("test", 0)
		if err := fl.Apply(tfs); err != nil {
			t.Error(err)
			return
		}

		output := fl.String()

		if output != test.expected {
			t.Errorf("%s does not match %s", output, test.expected)
		}
	}
}

func TestUintFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_BAR", "2")

	for _, test := range uintFlagTests {
		fl := &UintFlag{Name: test.name, Sources: EnvVars("APP_BAR")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_BAR"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%s does not end with"+expectedSuffix, output)
		}
	}
}

func TestUintFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Uint64("myflag", 42, "doc")
	cmd := &Command{flagSet: set}
	fl := &UintFlag{Name: "myflag"}
	require.Equal(t, uint64(42), fl.Get(cmd))
}

var uint64FlagTests = []struct {
	name     string
	expected string
}{
	{"gerfs", "--gerfs value\t(default: 8589934582)"},
	{"G", "-G value\t(default: 8589934582)"},
}

func TestUint64FlagHelpOutput(t *testing.T) {
	for _, test := range uint64FlagTests {
		fl := UintFlag{Name: test.name, Value: 8589934582}

		// create a temporary flag set to apply
		tfs := flag.NewFlagSet("test", 0)
		if err := fl.Apply(tfs); err != nil {
			t.Error(err)
			return
		}

		output := fl.String()

		if output != test.expected {
			t.Errorf("%s does not match %s", output, test.expected)
		}
	}
}

func TestUint64FlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_BAR", "2")

	for _, test := range uint64FlagTests {
		fl := &UintFlag{Name: test.name, Sources: EnvVars("APP_BAR")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_BAR"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%s does not end with"+expectedSuffix, output)
		}
	}
}

func TestUint64FlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Uint64("myflag", 42, "doc")
	cmd := &Command{flagSet: set}
	f := &UintFlag{Name: "myflag"}
	require.Equal(t, uint64(42), f.Get(cmd))
}

var durationFlagTests = []struct {
	name     string
	expected string
}{
	{"hooting", "--hooting value\t(default: 1s)"},
	{"H", "-H value\t(default: 1s)"},
}

func TestDurationFlagHelpOutput(t *testing.T) {
	for _, test := range durationFlagTests {
		fl := &DurationFlag{Name: test.name, Value: 1 * time.Second}

		// create a temporary flag set to apply
		tfs := flag.NewFlagSet("test", 0)
		if err := fl.Apply(tfs); err != nil {
			t.Error(err)
			return
		}

		output := fl.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestDurationFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_BAR", "2h3m6s")

	for _, test := range durationFlagTests {
		fl := &DurationFlag{Name: test.name, Sources: EnvVars("APP_BAR")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_BAR"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%s does not end with"+expectedSuffix, output)
		}
	}
}

func TestDurationFlagApply_SetsAllNames(t *testing.T) {
	v := time.Second * 20
	fl := DurationFlag{Name: "howmuch", Aliases: []string{"H", "whyyy"}, Destination: &v}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--howmuch", "30s", "-H", "5m", "--whyyy", "30h"})
	expect(t, err, nil)
	expect(t, v, time.Hour*30)
}

func TestDurationFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Duration("myflag", 42*time.Second, "doc")
	cmd := &Command{flagSet: set}
	f := &DurationFlag{Name: "myflag"}
	require.Equal(t, 42*time.Second, f.Get(cmd))
}

var intSliceFlagTests = []struct {
	name     string
	aliases  []string
	value    []int64
	expected string
}{
	{"heads", nil, []int64{}, "--heads value [ --heads value ]\t"},
	{"H", nil, []int64{}, "-H value [ -H value ]\t"},
	{"H", []string{"heads"}, []int64{9, 3}, "-H value, --heads value [ -H value, --heads value ]\t(default: 9, 3)"},
}

func TestIntSliceFlagHelpOutput(t *testing.T) {
	for _, test := range intSliceFlagTests {
		fl := &IntSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value}
		output := fl.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestIntSliceFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_SMURF", "42,3")

	for _, test := range intSliceFlagTests {
		fl := &IntSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value, Sources: EnvVars("APP_SMURF")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_SMURF"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%q does not end with"+expectedSuffix, output)
		}
	}
}

func TestIntSliceFlagApply_SetsAllNames(t *testing.T) {
	fl := IntSliceFlag{Name: "bits", Aliases: []string{"B", "bips"}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--bits", "23", "-B", "3", "--bips", "99"})
	expect(t, err, nil)
}

func TestIntSliceFlagApply_UsesEnvValues_noDefault(t *testing.T) {
	t.Setenv("MY_GOAT", "1 , 2")

	fl := &IntSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT")}
	set := flag.NewFlagSet("test", 0)

	r := require.New(t)
	r.NoError(fl.Apply(set))
	r.NoError(set.Parse(nil))
	r.Equal([]int64{1, 2}, set.Lookup("goat").Value.(flag.Getter).Get())
}

func TestIntSliceFlagApply_UsesEnvValues_withDefault(t *testing.T) {
	t.Setenv("MY_GOAT", "1 , 2")
	val := []int64{3, 4}
	fl := &IntSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val}
	set := flag.NewFlagSet("test", 0)

	r := require.New(t)
	r.NoError(fl.Apply(set))
	r.NoError(set.Parse(nil))
	r.Equal([]int64{3, 4}, val)
	r.Equal([]int64{1, 2}, set.Lookup("goat").Value.(flag.Getter).Get())
}

func TestIntSliceFlagApply_DefaultValueWithDestination(t *testing.T) {
	defValue := []int64{1, 2}
	dest := []int64{3}

	fl := IntSliceFlag{Name: "country", Value: defValue, Destination: &dest}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{})
	expect(t, err, nil)
	expect(t, defValue, dest)
}

func TestIntSliceFlagApply_ParentContext(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []int64{1, 2, 3}},
		},
		Commands: []*Command{
			{
				Name: "child",
				Action: func(_ context.Context, cmd *Command) error {
					require.Equalf(t, []int64{1, 2, 3}, cmd.IntSlice("numbers"), "child context unable to view parent flag")

					return nil
				},
			},
		},
	}).Run(buildTestContext(t), []string{"run", "child"})
}

func TestIntSliceFlag_SetFromParentCommand(t *testing.T) {
	fl := &IntSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []int64{1, 2, 3, 4}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)
	cmd := &Command{
		parent: &Command{
			flagSet: set,
		},
		flagSet: flag.NewFlagSet("empty", 0),
	}

	require.Equalf(t, []int64{1, 2, 3, 4}, cmd.IntSlice("numbers"), "child context unable to view parent flag")
}

func TestIntSliceFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Var(NewIntSlice(1, 2, 3), "myflag", "doc")
	cmd := &Command{flagSet: set}
	f := &IntSliceFlag{Name: "myflag"}
	require.Equal(t, []int64{1, 2, 3}, f.Get(cmd))
}

var uintSliceFlagTests = []struct {
	name     string
	aliases  []string
	value    []uint64
	expected string
}{
	{"heads", nil, []uint64{}, "--heads value [ --heads value ]\t"},
	{"H", nil, []uint64{}, "-H value [ -H value ]\t"},
	{
		"heads",
		[]string{"H"},
		[]uint64{2, 17179869184},
		"--heads value, -H value [ --heads value, -H value ]\t(default: 2, 17179869184)",
	},
}

func TestUintSliceFlagHelpOutput(t *testing.T) {
	for _, test := range uintSliceFlagTests {
		t.Run(test.name, func(t *testing.T) {
			fl := &UintSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value}
			require.Equal(t, test.expected, fl.String())
		})
	}
}

func TestUintSliceFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_SMURF", "42,17179869184")

	for _, test := range uintSliceFlagTests {
		fl := &UintSliceFlag{Name: test.name, Value: test.value, Sources: EnvVars("APP_SMURF")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_SMURF"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%q does not end with"+expectedSuffix, output)
		}
	}
}

func TestUintSliceFlagApply_SetsAllNames(t *testing.T) {
	fl := &UintSliceFlag{Name: "bits", Aliases: []string{"B", "bips"}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--bits", "23", "-B", "3", "--bips", "99"})
	expect(t, err, nil)
}

func TestUintSliceFlagApply_UsesEnvValues_noDefault(t *testing.T) {
	t.Setenv("MY_GOAT", "1 , 2")

	fl := &UintSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT")}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))

	r.NoError(set.Parse(nil))
	r.Equal([]uint64{1, 2}, set.Lookup("goat").Value.(flag.Getter).Get().([]uint64))
}

func TestUintSliceFlagApply_UsesEnvValues_withDefault(t *testing.T) {
	t.Setenv("MY_GOAT", "1 , 2")
	val := NewUintSlice(3, 4)
	fl := &UintSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val.Value()}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))
	r.NoError(set.Parse(nil))
	r.Equal([]uint64{3, 4}, val.Value())
	r.Equal([]uint64{1, 2}, set.Lookup("goat").Value.(flag.Getter).Get().([]uint64))
}

func TestUintSliceFlagApply_DefaultValueWithDestination(t *testing.T) {
	defValue := []uint64{1, 2}
	var dest []uint64

	fl := &UintSliceFlag{Name: "country", Value: defValue, Destination: &dest}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{})
	expect(t, err, nil)
	expect(t, defValue, dest)
}

func TestUintSliceFlagApply_ParentContext(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&UintSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []uint64{1, 2, 3}},
		},
		Commands: []*Command{
			{
				Name: "child",
				Action: func(_ context.Context, cmd *Command) error {
					require.Equalf(
						t, []uint64{1, 2, 3}, cmd.UintSlice("numbers"),
						"child context unable to view parent flag",
					)
					return nil
				},
			},
		},
	}).Run(buildTestContext(t), []string{"run", "child"})
}

func TestUintSliceFlag_SetFromParentCommand(t *testing.T) {
	fl := &UintSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []uint64{1, 2, 3, 4}}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))

	cmd := &Command{
		parent: &Command{
			flagSet: set,
		},
		flagSet: flag.NewFlagSet("empty", 0),
	}

	r.Equalf(
		[]uint64{1, 2, 3, 4},
		cmd.UintSlice("numbers"),
		"child context unable to view parent flag",
	)
}

func TestUintSliceFlag_ReturnNil(t *testing.T) {
	fl := &UintSliceFlag{}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))
	cmd := &Command{
		parent: &Command{
			flagSet: set,
		},
		flagSet: flag.NewFlagSet("empty", 0),
	}
	r.Equalf(
		[]uint64(nil),
		cmd.UintSlice("numbers"),
		"child context unable to view parent flag",
	)
}

var uint64SliceFlagTests = []struct {
	name     string
	aliases  []string
	value    []uint64
	expected string
}{
	{"heads", nil, []uint64{}, "--heads value [ --heads value ]\t"},
	{"H", nil, []uint64{}, "-H value [ -H value ]\t"},
	{
		"heads",
		[]string{"H"},
		[]uint64{2, 17179869184},
		"--heads value, -H value [ --heads value, -H value ]\t(default: 2, 17179869184)",
	},
}

func TestUint64SliceFlagHelpOutput(t *testing.T) {
	for _, test := range uint64SliceFlagTests {
		fl := UintSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value}
		output := fl.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestUint64SliceFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_SMURF", "42,17179869184")

	for _, test := range uint64SliceFlagTests {
		fl := UintSliceFlag{Name: test.name, Value: test.value, Sources: EnvVars("APP_SMURF")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_SMURF"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%q does not end with"+expectedSuffix, output)
		}
	}
}

func TestUint64SliceFlagApply_SetsAllNames(t *testing.T) {
	fl := UintSliceFlag{Name: "bits", Aliases: []string{"B", "bips"}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--bits", "23", "-B", "3", "--bips", "99"})
	expect(t, err, nil)
}

func TestUint64SliceFlagApply_UsesEnvValues_noDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "1 , 2")
	fl := UintSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT")}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get().([]uint64), []uint64{1, 2})
}

func TestUint64SliceFlagApply_UsesEnvValues_withDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "1 , 2")
	val := []uint64{3, 4}
	fl := UintSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)
	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get().([]uint64), []uint64{1, 2})
}

func TestUint64SliceFlagApply_DefaultValueWithDestination(t *testing.T) {
	defValue := []uint64{1, 2}
	dest := []uint64{3}

	fl := UintSliceFlag{Name: "country", Value: defValue, Destination: &dest}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{})
	expect(t, err, nil)
	expect(t, defValue, dest)
}

func TestUint64SliceFlagApply_ParentCommand(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&UintSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []uint64{1, 2, 3}},
		},
		Commands: []*Command{
			{
				Name: "child",
				Action: func(_ context.Context, cmd *Command) error {
					require.Equalf(
						t, []uint64{1, 2, 3}, cmd.UintSlice("numbers"),
						"child context unable to view parent flag",
					)
					return nil
				},
			},
		},
	}).Run(buildTestContext(t), []string{"run", "child"})
}

func TestUint64SliceFlag_SetFromParentCommand(t *testing.T) {
	fl := &UintSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []uint64{1, 2, 3, 4}}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))
	cmd := &Command{
		parent: &Command{
			flagSet: set,
		},
		flagSet: flag.NewFlagSet("empty", 0),
	}
	r.Equalf(
		[]uint64{1, 2, 3, 4}, cmd.UintSlice("numbers"),
		"child context unable to view parent flag",
	)
}

func TestUint64SliceFlag_ReturnNil(t *testing.T) {
	fl := &UintSliceFlag{}
	set := flag.NewFlagSet("test", 0)
	r := require.New(t)
	r.NoError(fl.Apply(set))
	cmd := &Command{
		parent: &Command{
			flagSet: set,
		},
		flagSet: flag.NewFlagSet("empty", 0),
	}
	r.Equalf(
		[]uint64(nil), cmd.UintSlice("numbers"),
		"child context unable to view parent flag",
	)
}

var float64FlagTests = []struct {
	name     string
	expected string
}{
	{"hooting", "--hooting value\t(default: 0.1)"},
	{"H", "-H value\t(default: 0.1)"},
}

func TestFloat64FlagHelpOutput(t *testing.T) {
	for _, test := range float64FlagTests {
		f := &FloatFlag{Name: test.name, Value: 0.1}
		output := f.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestFloat64FlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_BAZ", "99.4")

	for _, test := range float64FlagTests {
		fl := &FloatFlag{Name: test.name, Sources: EnvVars("APP_BAZ")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_BAZ"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%s does not end with"+expectedSuffix, output)
		}
	}
}

func TestFloat64FlagApply_SetsAllNames(t *testing.T) {
	v := 99.1
	fl := FloatFlag{Name: "noodles", Aliases: []string{"N", "nurbles"}, Destination: &v}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--noodles", "1.3", "-N", "11", "--nurbles", "43.33333"})
	expect(t, err, nil)
	expect(t, v, float64(43.33333))
}

func TestFloat64FlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Float64("myflag", 1.23, "doc")
	cmd := &Command{flagSet: set}
	f := &FloatFlag{Name: "myflag"}
	require.Equal(t, 1.23, f.Get(cmd))
}

var float64SliceFlagTests = []struct {
	name     string
	aliases  []string
	value    []float64
	expected string
}{
	{"heads", nil, []float64{}, "--heads value [ --heads value ]\t"},
	{"H", nil, []float64{}, "-H value [ -H value ]\t"},
	{
		"heads",
		[]string{"H"},
		[]float64{0.1234, -10.5},
		"--heads value, -H value [ --heads value, -H value ]\t(default: 0.1234, -10.5)",
	},
}

func TestFloat64SliceFlagHelpOutput(t *testing.T) {
	for _, test := range float64SliceFlagTests {
		fl := FloatSliceFlag{Name: test.name, Aliases: test.aliases, Value: test.value}
		output := fl.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestFloat64SliceFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_SMURF", "0.1234,-10.5")
	for _, test := range float64SliceFlagTests {
		fl := FloatSliceFlag{Name: test.name, Value: test.value, Sources: EnvVars("APP_SMURF")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_SMURF"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%q does not end with"+expectedSuffix, output)
		}
	}
}

func TestFloat64SliceFlagApply_SetsAllNames(t *testing.T) {
	fl := FloatSliceFlag{Name: "bits", Aliases: []string{"B", "bips"}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--bits", "23", "-B", "3", "--bips", "99"})
	expect(t, err, nil)
}

func TestFloat64SliceFlagApply_UsesEnvValues_noDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "1.0 , 2.0")

	fl := FloatSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT")}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get().([]float64), []float64{1, 2})
}

func TestFloat64SliceFlagApply_UsesEnvValues_withDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "1.0 , 2.0")
	val := []float64{3.0, 4.0}
	fl := FloatSliceFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)
	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get().([]float64), []float64{1, 2})
}

func TestFloat64SliceFlagApply_DefaultValueWithDestination(t *testing.T) {
	defValue := []float64{1.0, 2.0}
	dest := []float64{3}

	fl := FloatSliceFlag{Name: "country", Value: defValue, Destination: &dest}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{})
	expect(t, err, nil)
	expect(t, defValue, dest)
}

func TestFloat64SliceFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Var(NewFloatSlice(1.23, 4.56), "myflag", "doc")
	cmd := &Command{flagSet: set}
	f := &FloatSliceFlag{Name: "myflag"}
	require.Equal(t, []float64{1.23, 4.56}, f.Get(cmd))
}

func TestFloat64SliceFlagApply_ParentCommand(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&FloatSliceFlag{Name: "numbers", Aliases: []string{"n"}, Value: []float64{1.0, 2.0, 3.0}},
		},
		Commands: []*Command{
			{
				Name: "child",
				Action: func(_ context.Context, cmd *Command) error {
					require.Equalf(t, []float64{1.0, 2.0, 3.0}, cmd.FloatSlice("numbers"), "child context unable to view parent flag")
					return nil
				},
			},
		},
	}).Run(buildTestContext(t), []string{"run", "child"})
}

func TestParseMultiString(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&StringFlag{Name: "serve", Aliases: []string{"s"}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.String("serve") != "10" {
				t.Errorf("main name not set")
			}
			if cmd.String("s") != "10" {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10"})
}

func TestParseDestinationString(t *testing.T) {
	var dest string
	_ = (&Command{
		Flags: []Flag{
			&StringFlag{
				Name:        "dest",
				Destination: &dest,
			},
		},
		Action: func(context.Context, *Command) error {
			if dest != "10" {
				t.Errorf("expected destination String 10")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "--dest", "10"})
}

func TestParseMultiStringFromEnv(t *testing.T) {
	t.Setenv("APP_COUNT", "20")

	_ = (&Command{
		Flags: []Flag{
			&StringFlag{Name: "count", Aliases: []string{"c"}, Sources: EnvVars("APP_COUNT")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.String("count") != "20" {
				t.Errorf("main name not set")
			}
			if cmd.String("c") != "20" {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringFromEnvCascade(t *testing.T) {
	t.Setenv("APP_COUNT", "20")

	_ = (&Command{
		Flags: []Flag{
			&StringFlag{Name: "count", Aliases: []string{"c"}, Sources: EnvVars("COMPAT_COUNT", "APP_COUNT")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.String("count") != "20" {
				t.Errorf("main name not set")
			}
			if cmd.String("c") != "20" {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringSlice(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "serve", Aliases: []string{"s"}, Value: []string{}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			expected := []string{"10", "20"}
			if !reflect.DeepEqual(cmd.StringSlice("serve"), expected) {
				t.Errorf("main name not set: %v != %v", expected, cmd.StringSlice("serve"))
			}
			if !reflect.DeepEqual(cmd.StringSlice("s"), expected) {
				t.Errorf("short name not set: %v != %v", expected, cmd.StringSlice("s"))
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiStringSliceWithDefaults(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "serve", Aliases: []string{"s"}, Value: []string{"9", "2"}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			expected := []string{"10", "20"}
			if !reflect.DeepEqual(cmd.StringSlice("serve"), expected) {
				t.Errorf("main name not set: %v != %v", expected, cmd.StringSlice("serve"))
			}
			if !reflect.DeepEqual(cmd.StringSlice("s"), expected) {
				t.Errorf("short name not set: %v != %v", expected, cmd.StringSlice("s"))
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiStringSliceWithDestination(t *testing.T) {
	dest := []string{}

	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "serve", Aliases: []string{"s"}, Destination: &dest},
		},
		Action: func(_ context.Context, cmd *Command) error {
			expected := []string{"10", "20"}
			if !reflect.DeepEqual(dest, expected) {
				t.Errorf("main name not set: %v != %v", expected, cmd.StringSlice("serve"))
			}
			if !reflect.DeepEqual(dest, expected) {
				t.Errorf("short name not set: %v != %v", expected, cmd.StringSlice("s"))
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiStringSliceWithDestinationAndEnv(t *testing.T) {
	t.Setenv("APP_INTERVALS", "20,30,40")

	dest := []string{}
	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "serve", Aliases: []string{"s"}, Destination: &dest, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			expected := []string{"10", "20"}
			if !reflect.DeepEqual(dest, expected) {
				t.Errorf("main name not set: %v != %v", expected, cmd.StringSlice("serve"))
			}
			if !reflect.DeepEqual(dest, expected) {
				t.Errorf("short name not set: %v != %v", expected, cmd.StringSlice("s"))
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiFloat64SliceWithDestinationAndEnv(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	dest := []float64{}
	_ = (&Command{
		Flags: []Flag{
			&FloatSliceFlag{Name: "serve", Aliases: []string{"s"}, Destination: &dest, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			expected := []float64{10, 20}
			if !reflect.DeepEqual(dest, expected) {
				t.Errorf("main name not set: %v != %v", expected, cmd.StringSlice("serve"))
			}
			if !reflect.DeepEqual(dest, expected) {
				t.Errorf("short name not set: %v != %v", expected, cmd.StringSlice("s"))
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiIntSliceWithDestinationAndEnv(t *testing.T) {
	t.Setenv("APP_INTERVALS", "20,30,40")

	dest := []int64{}
	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "serve", Aliases: []string{"s"}, Destination: &dest, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(context.Context, *Command) error {
			require.Equalf(t, []int64{10, 20}, dest, "main name not set")

			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiStringSliceWithDefaultsUnset(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "serve", Aliases: []string{"s"}, Value: []string{"9", "2"}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.StringSlice("serve"), []string{"9", "2"}) {
				t.Errorf("main name not set: %v", cmd.StringSlice("serve"))
			}
			if !reflect.DeepEqual(cmd.StringSlice("s"), []string{"9", "2"}) {
				t.Errorf("short name not set: %v", cmd.StringSlice("s"))
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringSliceFromEnv(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []string{}, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.StringSlice("intervals"), []string{"20", "30", "40"}) {
				t.Errorf("main name not set from env")
			}
			if !reflect.DeepEqual(cmd.StringSlice("i"), []string{"20", "30", "40"}) {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringSliceFromEnvWithDefaults(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []string{"1", "2", "5"}, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.StringSlice("intervals"), []string{"20", "30", "40"}) {
				t.Errorf("main name not set from env")
			}
			if !reflect.DeepEqual(cmd.StringSlice("i"), []string{"20", "30", "40"}) {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringSliceFromEnvCascade(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []string{}, Sources: EnvVars("COMPAT_INTERVALS", "APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.StringSlice("intervals"), []string{"20", "30", "40"}) {
				t.Errorf("main name not set from env")
			}
			if !reflect.DeepEqual(cmd.StringSlice("i"), []string{"20", "30", "40"}) {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringSliceFromEnvCascadeWithDefaults(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []string{"1", "2", "5"}, Sources: EnvVars("COMPAT_INTERVALS", "APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.StringSlice("intervals"), []string{"20", "30", "40"}) {
				t.Errorf("main name not set from env")
			}
			if !reflect.DeepEqual(cmd.StringSlice("i"), []string{"20", "30", "40"}) {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiStringSliceFromEnvWithDestination(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	dest := []string{}
	_ = (&Command{
		Flags: []Flag{
			&StringSliceFlag{Name: "intervals", Aliases: []string{"i"}, Destination: &dest, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(context.Context, *Command) error {
			if !reflect.DeepEqual(dest, []string{"20", "30", "40"}) {
				t.Errorf("main name not set from env")
			}
			if !reflect.DeepEqual(dest, []string{"20", "30", "40"}) {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiInt(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&IntFlag{Name: "serve", Aliases: []string{"s"}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Int("serve") != 10 {
				t.Errorf("main name not set")
			}
			if cmd.Int("s") != 10 {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10"})
}

func TestParseDestinationInt(t *testing.T) {
	var dest int64
	_ = (&Command{
		Flags: []Flag{
			&IntFlag{
				Name:        "dest",
				Destination: &dest,
			},
		},
		Action: func(context.Context, *Command) error {
			if dest != 10 {
				t.Errorf("expected destination Int 10")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "--dest", "10"})
}

func TestParseMultiIntFromEnv(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_TIMEOUT_SECONDS", "10")
	_ = (&Command{
		Flags: []Flag{
			&IntFlag{Name: "timeout", Aliases: []string{"t"}, Sources: EnvVars("APP_TIMEOUT_SECONDS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Int("timeout") != 10 {
				t.Errorf("main name not set")
			}
			if cmd.Int("t") != 10 {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiIntFromEnvCascade(t *testing.T) {
	t.Setenv("APP_TIMEOUT_SECONDS", "10")
	_ = (&Command{
		Flags: []Flag{
			&IntFlag{Name: "timeout", Aliases: []string{"t"}, Sources: EnvVars("COMPAT_TIMEOUT_SECONDS", "APP_TIMEOUT_SECONDS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Int("timeout") != 10 {
				t.Errorf("main name not set")
			}
			if cmd.Int("t") != 10 {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiIntSlice(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "serve", Aliases: []string{"s"}, Value: []int64{}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			r := require.New(t)

			r.Equalf([]int64{10, 20}, cmd.IntSlice("serve"), "main name not set")
			r.Equalf([]int64{10, 20}, cmd.IntSlice("s"), "short name not set")

			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiIntSliceWithDefaults(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "serve", Aliases: []string{"s"}, Value: []int64{9, 2}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			r := require.New(t)

			r.Equalf([]int64{10, 20}, cmd.IntSlice("serve"), "main name not set")
			r.Equalf([]int64{10, 20}, cmd.IntSlice("s"), "short name not set")

			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10", "-s", "20"})
}

func TestParseMultiIntSliceWithDefaultsUnset(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "serve", Aliases: []string{"s"}, Value: []int64{9, 2}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.IntSlice("serve"), []int64{9, 2}) {
				t.Errorf("main name not set")
			}
			if !reflect.DeepEqual(cmd.IntSlice("s"), []int64{9, 2}) {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiIntSliceFromEnv(t *testing.T) {
	t.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []int64{}, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			r := require.New(t)

			r.Equalf([]int64{20, 30, 40}, cmd.IntSlice("intervals"), "main name not set from env")
			r.Equalf([]int64{20, 30, 40}, cmd.IntSlice("i"), "short name not set from env")

			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiIntSliceFromEnvWithDefaults(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []int64{1, 2, 5}, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if !reflect.DeepEqual(cmd.IntSlice("intervals"), []int64{20, 30, 40}) {
				t.Errorf("main name not set from env")
			}
			if !reflect.DeepEqual(cmd.IntSlice("i"), []int64{20, 30, 40}) {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiIntSliceFromEnvCascade(t *testing.T) {
	t.Setenv("APP_INTERVALS", "20,30,40")

	_ = (&Command{
		Flags: []Flag{
			&IntSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []int64{}, Sources: EnvVars("COMPAT_INTERVALS", "APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			r := require.New(t)

			r.Equalf([]int64{20, 30, 40}, cmd.IntSlice("intervals"), "main name not set from env")
			r.Equalf([]int64{20, 30, 40}, cmd.IntSlice("i"), "short name not set from env")

			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiFloat64(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&FloatFlag{Name: "serve", Aliases: []string{"s"}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Float("serve") != 10.2 {
				t.Errorf("main name not set")
			}
			if cmd.Float("s") != 10.2 {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "-s", "10.2"})
}

func TestParseDestinationFloat64(t *testing.T) {
	var dest float64
	_ = (&Command{
		Flags: []Flag{
			&FloatFlag{
				Name:        "dest",
				Destination: &dest,
			},
		},
		Action: func(context.Context, *Command) error {
			if dest != 10.2 {
				t.Errorf("expected destination Float64 10.2")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "--dest", "10.2"})
}

func TestParseMultiFloat64FromEnv(t *testing.T) {
	t.Setenv("APP_TIMEOUT_SECONDS", "15.5")
	_ = (&Command{
		Flags: []Flag{
			&FloatFlag{Name: "timeout", Aliases: []string{"t"}, Sources: EnvVars("APP_TIMEOUT_SECONDS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Float("timeout") != 15.5 {
				t.Errorf("main name not set")
			}
			if cmd.Float("t") != 15.5 {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiFloat64FromEnvCascade(t *testing.T) {
	t.Setenv("APP_TIMEOUT_SECONDS", "15.5")

	_ = (&Command{
		Flags: []Flag{
			&FloatFlag{Name: "timeout", Aliases: []string{"t"}, Sources: EnvVars("COMPAT_TIMEOUT_SECONDS", "APP_TIMEOUT_SECONDS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Float("timeout") != 15.5 {
				t.Errorf("main name not set")
			}
			if cmd.Float("t") != 15.5 {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiFloat64SliceFromEnv(t *testing.T) {
	t.Setenv("APP_INTERVALS", "0.1,-10.5")

	_ = (&Command{
		Flags: []Flag{
			&FloatSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []float64{}, Sources: EnvVars("APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			require.Equalf(t, []float64{0.1, -10.5}, cmd.FloatSlice("intervals"), "main name not set from env")
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiFloat64SliceFromEnvCascade(t *testing.T) {
	t.Setenv("APP_INTERVALS", "0.1234,-10.5")

	_ = (&Command{
		Flags: []Flag{
			&FloatSliceFlag{Name: "intervals", Aliases: []string{"i"}, Value: []float64{}, Sources: EnvVars("COMPAT_INTERVALS", "APP_INTERVALS")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			require.Equalf(t, []float64{0.1234, -10.5}, cmd.FloatSlice("intervals"), "main name not set from env")
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiBool(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&BoolFlag{Name: "serve", Aliases: []string{"s"}},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Bool("serve") != true {
				t.Errorf("main name not set")
			}
			if cmd.Bool("s") != true {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "--serve"})
}

func TestParseBoolShortOptionHandle(t *testing.T) {
	_ = (&Command{
		Commands: []*Command{
			{
				Name:                   "foobar",
				UseShortOptionHandling: true,
				Action: func(_ context.Context, cmd *Command) error {
					if cmd.Bool("serve") != true {
						t.Errorf("main name not set")
					}
					if cmd.Bool("option") != true {
						t.Errorf("short name not set")
					}
					return nil
				},
				Flags: []Flag{
					&BoolFlag{Name: "serve", Aliases: []string{"s"}},
					&BoolFlag{Name: "option", Aliases: []string{"o"}},
				},
			},
		},
	}).Run(buildTestContext(t), []string{"run", "foobar", "-so"})
}

func TestParseDestinationBool(t *testing.T) {
	var dest bool
	_ = (&Command{
		Flags: []Flag{
			&BoolFlag{
				Name:        "dest",
				Destination: &dest,
			},
		},
		Action: func(context.Context, *Command) error {
			if dest != true {
				t.Errorf("expected destination Bool true")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "--dest"})
}

func TestParseMultiBoolFromEnv(t *testing.T) {
	t.Setenv("APP_DEBUG", "1")
	_ = (&Command{
		Flags: []Flag{
			&BoolFlag{Name: "debug", Aliases: []string{"d"}, Sources: EnvVars("APP_DEBUG")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Bool("debug") != true {
				t.Errorf("main name not set from env")
			}
			if cmd.Bool("d") != true {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseMultiBoolFromEnvCascade(t *testing.T) {
	t.Setenv("APP_DEBUG", "1")
	_ = (&Command{
		Flags: []Flag{
			&BoolFlag{Name: "debug", Aliases: []string{"d"}, Sources: EnvVars("COMPAT_DEBUG", "APP_DEBUG")},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Bool("debug") != true {
				t.Errorf("main name not set from env")
			}
			if cmd.Bool("d") != true {
				t.Errorf("short name not set from env")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run"})
}

func TestParseBoolFromEnv(t *testing.T) {
	boolFlagTests := []struct {
		input  string
		output bool
	}{
		{"", false},
		{"1", true},
		{"false", false},
		{"true", true},
	}

	for _, test := range boolFlagTests {
		t.Run(fmt.Sprintf("%[1]q %[2]v", test.input, test.output), func(t *testing.T) {
			t.Setenv("DEBUG", test.input)
			_ = (&Command{
				Flags: []Flag{
					&BoolFlag{Name: "debug", Aliases: []string{"d"}, Sources: EnvVars("DEBUG")},
				},
				Action: func(_ context.Context, cmd *Command) error {
					if cmd.Bool("debug") != test.output {
						t.Errorf("expected %+v to be parsed as %+v, instead was %+v", test.input, test.output, cmd.Bool("debug"))
					}
					if cmd.Bool("d") != test.output {
						t.Errorf("expected %+v to be parsed as %+v, instead was %+v", test.input, test.output, cmd.Bool("d"))
					}
					return nil
				},
			}).Run(buildTestContext(t), []string{"run"})
		})
	}
}

func TestParseMultiBoolT(t *testing.T) {
	_ = (&Command{
		Flags: []Flag{
			&BoolFlag{Name: "implode", Aliases: []string{"i"}, Value: true},
		},
		Action: func(_ context.Context, cmd *Command) error {
			if cmd.Bool("implode") {
				t.Errorf("main name not set")
			}
			if cmd.Bool("i") {
				t.Errorf("short name not set")
			}
			return nil
		},
	}).Run(buildTestContext(t), []string{"run", "--implode=false"})
}

type Parser [2]string

func (p *Parser) Set(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format")
	}

	(*p)[0] = parts[0]
	(*p)[1] = parts[1]

	return nil
}

func (p *Parser) String() string {
	return fmt.Sprintf("%s,%s", p[0], p[1])
}

func (p *Parser) Get() interface{} {
	return p
}

func TestStringSlice_Serialized_Set(t *testing.T) {
	sl0 := NewStringSlice("a", "b")
	ser0 := sl0.Serialize()

	if len(ser0) < len(slPfx) {
		t.Fatalf("serialized shorter than expected: %q", ser0)
	}

	sl1 := NewStringSlice("c", "d")
	_ = sl1.Set(ser0)

	if sl0.String() != sl1.String() {
		t.Fatalf("pre and post serialization do not match: %v != %v", sl0, sl1)
	}
}

func TestIntSlice_Serialized_Set(t *testing.T) {
	sl0 := NewIntSlice(1, 2)
	ser0 := sl0.Serialize()

	if len(ser0) < len(slPfx) {
		t.Fatalf("serialized shorter than expected: %q", ser0)
	}

	sl1 := NewIntSlice(3, 4)
	_ = sl1.Set(ser0)

	if sl0.String() != sl1.String() {
		t.Fatalf("pre and post serialization do not match: %v != %v", sl0, sl1)
	}
}

func TestUintSlice_Serialized_Set(t *testing.T) {
	sl0 := NewUintSlice(1, 2)
	ser0 := sl0.Serialize()

	if len(ser0) < len(slPfx) {
		t.Fatalf("serialized shorter than expected: %q", ser0)
	}

	sl1 := NewUintSlice(3, 4)
	_ = sl1.Set(ser0)

	if sl0.String() != sl1.String() {
		t.Fatalf("pre and post serialization do not match: %v != %v", sl0, sl1)
	}
}

func TestUint64Slice_Serialized_Set(t *testing.T) {
	sl0 := NewUintSlice(1, 2)
	ser0 := sl0.Serialize()

	if len(ser0) < len(slPfx) {
		t.Fatalf("serialized shorter than expected: %q", ser0)
	}

	sl1 := NewUintSlice(3, 4)
	_ = sl1.Set(ser0)

	if sl0.String() != sl1.String() {
		t.Fatalf("pre and post serialization do not match: %v != %v", sl0, sl1)
	}
}

func TestStringMap_Serialized_Set(t *testing.T) {
	m0 := NewStringMap(map[string]string{"a": "b"})
	ser0 := m0.Serialize()

	if len(ser0) < len(slPfx) {
		t.Fatalf("serialized shorter than expected: %q", ser0)
	}

	m1 := NewStringMap(map[string]string{"c": "d"})
	_ = m1.Set(ser0)

	if m0.String() != m1.String() {
		t.Fatalf("pre and post serialization do not match: %v != %v", m0, m1)
	}
}

func TestTimestamp_set(t *testing.T) {
	ts := timestampValue{
		timestamp:  nil,
		hasBeenSet: false,
		layout:     "Jan 2, 2006 at 3:04pm (MST)",
	}

	time1 := "Feb 3, 2013 at 7:54pm (PST)"
	if err := ts.Set(time1); err != nil {
		t.Fatalf("Failed to parse time %s with layout %s", time1, ts.layout)
	}
	if ts.hasBeenSet == false {
		t.Fatalf("hasBeenSet is not true after setting a time")
	}

	ts.hasBeenSet = false
	ts.layout = time.RFC3339
	time2 := "2006-01-02T15:04:05Z"
	if err := ts.Set(time2); err != nil {
		t.Fatalf("Failed to parse time %s with layout %s", time2, ts.layout)
	}
	if ts.hasBeenSet == false {
		t.Fatalf("hasBeenSet is not true after setting a time")
	}
}

func TestTimestampFlagApply(t *testing.T) {
	expectedResult, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	fl := TimestampFlag{Name: "time", Aliases: []string{"t"}, Config: TimestampConfig{Layout: time.RFC3339}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--time", "2006-01-02T15:04:05Z"})
	expect(t, err, nil)
	expect(t, set.Lookup("time").Value.(flag.Getter).Get(), expectedResult)
}

func TestTimestampFlagApplyValue(t *testing.T) {
	expectedResult, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	fl := TimestampFlag{Name: "time", Aliases: []string{"t"}, Config: TimestampConfig{Layout: time.RFC3339}, Value: expectedResult}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{""})
	expect(t, err, nil)
	expect(t, set.Lookup("time").Value.(flag.Getter).Get(), expectedResult)
}

func TestTimestampFlagApply_Fail_Parse_Wrong_Layout(t *testing.T) {
	fl := TimestampFlag{Name: "time", Aliases: []string{"t"}, Config: TimestampConfig{Layout: "randomlayout"}}
	set := flag.NewFlagSet("test", 0)
	set.SetOutput(io.Discard)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--time", "2006-01-02T15:04:05Z"})
	expect(t, err, fmt.Errorf("invalid value \"2006-01-02T15:04:05Z\" for flag -time: parsing time \"2006-01-02T15:04:05Z\" as \"randomlayout\": cannot parse \"2006-01-02T15:04:05Z\" as \"randomlayout\""))
}

func TestTimestampFlagApply_Fail_Parse_Wrong_Time(t *testing.T) {
	fl := TimestampFlag{Name: "time", Aliases: []string{"t"}, Config: TimestampConfig{Layout: "Jan 2, 2006 at 3:04pm (MST)"}}
	set := flag.NewFlagSet("test", 0)
	set.SetOutput(io.Discard)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--time", "2006-01-02T15:04:05Z"})
	expect(t, err, fmt.Errorf("invalid value \"2006-01-02T15:04:05Z\" for flag -time: parsing time \"2006-01-02T15:04:05Z\" as \"Jan 2, 2006 at 3:04pm (MST)\": cannot parse \"2006-01-02T15:04:05Z\" as \"Jan\""))
}

func TestTimestampFlagApply_Timezoned(t *testing.T) {
	pdt := time.FixedZone("PDT", -7*60*60)
	expectedResult, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	fl := TimestampFlag{Name: "time", Aliases: []string{"t"}, Config: TimestampConfig{Layout: time.ANSIC, Timezone: pdt}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--time", "Mon Jan 2 08:04:05 2006"})
	expect(t, err, nil)
	expect(t, set.Lookup("time").Value.(flag.Getter).Get(), expectedResult.In(pdt))
}

func TestTimestampFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	now := time.Now()
	set.Var(newTimestamp(now), "myflag", "doc")
	cmd := &Command{flagSet: set}
	f := &TimestampFlag{Name: "myflag"}
	require.Equal(t, now, f.Get(cmd))
}

type flagDefaultTestCase struct {
	name    string
	flag    Flag
	toParse []string
	expect  string
}

func TestFlagDefaultValue(t *testing.T) {
	cases := []*flagDefaultTestCase{
		{
			name:    "stringSlice",
			flag:    &StringSliceFlag{Name: "flag", Value: []string{"default1", "default2"}},
			toParse: []string{"--flag", "parsed"},
			expect:  `--flag value [ --flag value ]	(default: "default1", "default2")`,
		},
		{
			name:    "float64Slice",
			flag:    &FloatSliceFlag{Name: "flag", Value: []float64{1.1, 2.2}},
			toParse: []string{"--flag", "13.3"},
			expect:  `--flag value [ --flag value ]	(default: 1.1, 2.2)`,
		},
		{
			name:    "intSlice",
			flag:    &IntSliceFlag{Name: "flag", Value: []int64{1, 2}},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value [ --flag value ]	(default: 1, 2)`,
		},
		{
			name:    "uintSlice",
			flag:    &UintSliceFlag{Name: "flag", Value: []uint64{1, 2}},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value [ --flag value ]	(default: 1, 2)`,
		},
		{
			name:    "string",
			flag:    &StringFlag{Name: "flag", Value: "default"},
			toParse: []string{"--flag", "parsed"},
			expect:  `--flag value	(default: "default")`,
		},
		{
			name:    "bool",
			flag:    &BoolFlag{Name: "flag", Value: true},
			toParse: []string{"--flag", "false"},
			expect:  `--flag	(default: true)`,
		},
		{
			name:    "uint64",
			flag:    &UintFlag{Name: "flag", Value: 1},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value	(default: 1)`,
		},
		{
			name:    "stringMap",
			flag:    &StringMapFlag{Name: "flag", Value: map[string]string{"default1": "default2"}},
			toParse: []string{"--flag", "parsed="},
			expect:  `--flag value [ --flag value ]	(default: default1="default2")`,
		},
	}
	for i, v := range cases {
		set := flag.NewFlagSet("test", 0)
		set.SetOutput(io.Discard)
		_ = v.flag.Apply(set)
		if err := set.Parse(v.toParse); err != nil {
			t.Error(err)
		}
		if got := v.flag.String(); got != v.expect {
			t.Errorf("TestFlagDefaultValue %d %s\nexpect:%s\ngot:%s", i, v.name, v.expect, got)
		}
	}
}

type flagDefaultTestCaseWithEnv struct {
	name    string
	flag    Flag
	toParse []string
	expect  string
	environ map[string]string
}

func TestFlagDefaultValueWithEnv(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()

	ts, err := time.Parse(time.RFC3339, "2005-01-02T15:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	cases := []*flagDefaultTestCaseWithEnv{
		{
			name:    "stringSlice",
			flag:    &StringSliceFlag{Name: "flag", Value: []string{"default1", "default2"}, Sources: EnvVars("ssflag")},
			toParse: []string{"--flag", "parsed"},
			expect:  `--flag value [ --flag value ]	(default: "default1", "default2")` + withEnvHint([]string{"ssflag"}, ""),
			environ: map[string]string{
				"ssflag": "some-other-env_value",
			},
		},
		{
			name:    "float64Slice",
			flag:    &FloatSliceFlag{Name: "flag", Value: []float64{1.1, 2.2}, Sources: EnvVars("fsflag")},
			toParse: []string{"--flag", "13.3"},
			expect:  `--flag value [ --flag value ]	(default: 1.1, 2.2)` + withEnvHint([]string{"fsflag"}, ""),
			environ: map[string]string{
				"fsflag": "20304.222",
			},
		},
		{
			name:    "intSlice",
			flag:    &IntSliceFlag{Name: "flag", Value: []int64{1, 2}, Sources: EnvVars("isflag")},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value [ --flag value ]	(default: 1, 2)` + withEnvHint([]string{"isflag"}, ""),
			environ: map[string]string{
				"isflag": "101",
			},
		},
		{
			name:    "uintSlice",
			flag:    &UintSliceFlag{Name: "flag", Value: []uint64{1, 2}, Sources: EnvVars("uisflag")},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value [ --flag value ]	(default: 1, 2)` + withEnvHint([]string{"uisflag"}, ""),
			environ: map[string]string{
				"uisflag": "3",
			},
		},
		{
			name:    "string",
			flag:    &StringFlag{Name: "flag", Value: "default", Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "parsed"},
			expect:  `--flag value	(default: "default")` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "some-other-string",
			},
		},
		{
			name:    "bool",
			flag:    &BoolFlag{Name: "flag", Value: true, Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "false"},
			expect:  `--flag	(default: true)` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "false",
			},
		},
		{
			name:    "uint64",
			flag:    &UintFlag{Name: "flag", Value: 1, Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value	(default: 1)` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "10",
			},
		},
		{
			name:    "uint",
			flag:    &UintFlag{Name: "flag", Value: 1, Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value	(default: 1)` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "10",
			},
		},
		{
			name:    "int64",
			flag:    &IntFlag{Name: "flag", Value: 1, Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value	(default: 1)` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "10",
			},
		},
		{
			name:    "int",
			flag:    &IntFlag{Name: "flag", Value: 1, Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "13"},
			expect:  `--flag value	(default: 1)` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "10",
			},
		},
		{
			name:    "duration",
			flag:    &DurationFlag{Name: "flag", Value: time.Second, Sources: EnvVars("uflag")},
			toParse: []string{"--flag", "2m"},
			expect:  `--flag value	(default: 1s)` + withEnvHint([]string{"uflag"}, ""),
			environ: map[string]string{
				"uflag": "2h4m10s",
			},
		},
		{
			name:    "timestamp",
			flag:    &TimestampFlag{Name: "flag", Value: ts, Config: TimestampConfig{Layout: time.RFC3339}, Sources: EnvVars("tflag")},
			toParse: []string{"--flag", "2006-11-02T15:04:05Z"},
			expect:  `--flag value	(default: 2005-01-02 15:04:05 +0000 UTC)` + withEnvHint([]string{"tflag"}, ""),
			environ: map[string]string{
				"tflag": "2010-01-02T15:04:05Z",
			},
		},
		{
			name:    "stringMap",
			flag:    &StringMapFlag{Name: "flag", Value: map[string]string{"default1": "default2"}, Sources: EnvVars("ssflag")},
			toParse: []string{"--flag", "parsed="},
			expect:  `--flag value [ --flag value ]	(default: default1="default2")` + withEnvHint([]string{"ssflag"}, ""),
			environ: map[string]string{
				"ssflag": "some-other-env_value=",
			},
		},
	}
	for i, v := range cases {
		for key, val := range v.environ {
			os.Setenv(key, val)
		}
		set := flag.NewFlagSet("test", 0)
		set.SetOutput(io.Discard)
		if err := v.flag.Apply(set); err != nil {
			t.Fatal(err)
		}
		if err := set.Parse(v.toParse); err != nil {
			t.Error(err)
		}
		if got := v.flag.String(); got != v.expect {
			t.Errorf("TestFlagDefaultValue %d %s\nexpect:%s\ngot:%s", i, v.name, v.expect, got)
		}
	}
}

type flagValueTestCase struct {
	name    string
	flag    Flag
	toParse []string
	expect  string
}

func TestFlagValue(t *testing.T) {
	cases := []*flagValueTestCase{
		{
			name:    "stringSlice",
			flag:    &StringSliceFlag{Name: "flag", Value: []string{"default1", "default2"}},
			toParse: []string{"--flag", "parsed,parsed2", "--flag", "parsed3,parsed4"},
			expect:  `[parsed parsed2 parsed3 parsed4]`,
		},
		{
			name:    "float64Slice",
			flag:    &FloatSliceFlag{Name: "flag", Value: []float64{1.1, 2.2}},
			toParse: []string{"--flag", "13.3,14.4", "--flag", "15.5,16.6"},
			expect:  `[]float64{13.3, 14.4, 15.5, 16.6}`,
		},
		{
			name:    "intSlice",
			flag:    &IntSliceFlag{Name: "flag", Value: []int64{1, 2}},
			toParse: []string{"--flag", "13,14", "--flag", "15,16"},
			expect:  `[]int64{13, 14, 15, 16}`,
		},
		{
			name:    "uintSlice",
			flag:    &UintSliceFlag{Name: "flag", Value: []uint64{1, 2}},
			toParse: []string{"--flag", "13,14", "--flag", "15,16"},
			expect:  `[]uint64{13, 14, 15, 16}`,
		},
		{
			name:    "stringMap",
			flag:    &StringMapFlag{Name: "flag", Value: map[string]string{"default1": "default2"}},
			toParse: []string{"--flag", "parsed=parsed2", "--flag", "parsed3=parsed4"},
			expect:  `map[parsed:parsed2 parsed3:parsed4]`,
		},
	}
	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			set := flag.NewFlagSet("test", 0)
			set.SetOutput(io.Discard)
			_ = v.flag.Apply(set)
			if err := set.Parse(v.toParse); err != nil {
				t.Error(err)
			}
			f := set.Lookup("flag")
			require.Equal(t, v.expect, f.Value.String())
		})
	}
}

func TestTimestampFlagApply_WithDestination(t *testing.T) {
	var destination time.Time
	expectedResult, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	fl := TimestampFlag{Name: "time", Aliases: []string{"t"}, Config: TimestampConfig{Layout: time.RFC3339}, Destination: &destination}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--time", "2006-01-02T15:04:05Z"})
	expect(t, err, nil)
	expect(t, destination, expectedResult)
}

// Test issue #1254
// StringSlice() with UseShortOptionHandling causes duplicated entries, depending on the ordering of the flags
func TestSliceShortOptionHandle(t *testing.T) {
	wasCalled := false
	err := (&Command{
		Name:                   "foobar",
		UseShortOptionHandling: true,
		Action: func(_ context.Context, cmd *Command) error {
			wasCalled = true

			if !cmd.Bool("i") {
				return fmt.Errorf("bool i not set")
			}

			if !cmd.Bool("t") {
				return fmt.Errorf("bool i not set")
			}

			ss := cmd.StringSlice("net")
			if !reflect.DeepEqual(ss, []string{"foo"}) {
				return fmt.Errorf("got different slice %q than expected", ss)
			}

			return nil
		},
		Flags: []Flag{
			&StringSliceFlag{Name: "net"},
			&BoolFlag{Name: "i"},
			&BoolFlag{Name: "t"},
		},
	}).Run(buildTestContext(t), []string{"foobar", "--net=foo", "-it"})

	r := require.New(t)

	r.NoError(err)
	r.Truef(wasCalled, "action callback was never called")
}

// Test issue #1541
func TestCustomizedSliceFlagSeparator(t *testing.T) {
	defaultSliceFlagSeparator = ";"
	defer func() {
		defaultSliceFlagSeparator = ","
	}()
	opts := []string{"opt1", "opt2", "opt3,op", "opt4"}
	ret := flagSplitMultiValues(strings.Join(opts, ";"))
	if len(ret) != 4 {
		t.Fatalf("split slice flag failed, want: 4, but get: %d", len(ret))
	}
	for idx, r := range ret {
		if r != opts[idx] {
			t.Fatalf("get %dth failed, wanted: %s, but get: %s", idx, opts[idx], r)
		}
	}
}

func TestFlagSplitMultiValues_Disabled(t *testing.T) {
	disableSliceFlagSeparator = true
	defer func() {
		disableSliceFlagSeparator = false
	}()

	opts := []string{"opt1", "opt2", "opt3,op", "opt4"}
	ret := flagSplitMultiValues(strings.Join(opts, defaultSliceFlagSeparator))
	if len(ret) != 1 {
		t.Fatalf("failed to disable split slice flag, want: 1, but got: %d", len(ret))
	}

	if ret[0] != strings.Join(opts, defaultSliceFlagSeparator) {
		t.Fatalf("failed to disable split slice flag, want: %s, but got: %s", strings.Join(opts, defaultSliceFlagSeparator), ret[0])
	}
}

var stringMapFlagTests = []struct {
	name     string
	aliases  []string
	value    map[string]string
	expected string
}{
	{"foo", nil, nil, "--foo value [ --foo value ]\t"},
	{"f", nil, nil, "-f value [ -f value ]\t"},
	{"f", nil, map[string]string{"Lipstick": ""}, "-f value [ -f value ]\t(default: Lipstick=)"},
	{"test", nil, map[string]string{"Something": ""}, "--test value [ --test value ]\t(default: Something=)"},
	{"dee", []string{"d"}, map[string]string{"Inka": "Dinka", "dooo": ""}, "--dee value, -d value [ --dee value, -d value ]\t(default: Inka=\"Dinka\", dooo=)"},
}

func TestStringMapFlagHelpOutput(t *testing.T) {
	for _, test := range stringMapFlagTests {
		f := &StringMapFlag{Name: test.name, Aliases: test.aliases, Value: test.value}
		output := f.String()

		if output != test.expected {
			t.Errorf("%q does not match %q", output, test.expected)
		}
	}
}

func TestStringMapFlagWithEnvVarHelpOutput(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("APP_QWWX", "11,4")

	for _, test := range stringMapFlagTests {
		fl := &StringMapFlag{Name: test.name, Aliases: test.aliases, Value: test.value, Sources: EnvVars("APP_QWWX")}
		output := fl.String()

		expectedSuffix := withEnvHint([]string{"APP_QWWX"}, "")
		if !strings.HasSuffix(output, expectedSuffix) {
			t.Errorf("%q does not end with"+expectedSuffix, output)
		}
	}
}

func TestStringMapFlagApply_SetsAllNames(t *testing.T) {
	fl := StringMapFlag{Name: "goat", Aliases: []string{"G", "gooots"}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--goat", "aaa=", "-G", "bbb=", "--gooots", "eeeee="})
	expect(t, err, nil)
}

func TestStringMapFlagApply_UsesEnvValues_noDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "vincent van goat=scape goat")
	var val map[string]string
	fl := StringMapFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, val, map[string]string(nil))
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get(), map[string]string{"vincent van goat": "scape goat"})
}

func TestStringMapFlagApply_UsesEnvValues_withDefault(t *testing.T) {
	defer resetEnv(os.Environ())
	os.Clearenv()
	_ = os.Setenv("MY_GOAT", "vincent van goat=scape goat")
	val := map[string]string{`some default`: `values here`}
	fl := StringMapFlag{Name: "goat", Sources: EnvVars("MY_GOAT"), Value: val}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)
	err := set.Parse(nil)
	expect(t, err, nil)
	expect(t, val, map[string]string{`some default`: `values here`})
	expect(t, set.Lookup("goat").Value.(flag.Getter).Get(), map[string]string{"vincent van goat": "scape goat"})
}

func TestStringMapFlagApply_DefaultValueWithDestination(t *testing.T) {
	defValue := map[string]string{"UA": "US"}

	fl := StringMapFlag{Name: "country", Value: defValue, Destination: &map[string]string{"CA": ""}}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{})
	expect(t, err, nil)
	expect(t, defValue, *fl.Destination)
}

func TestStringMapFlagValueFromCommand(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.Var(NewStringMap(map[string]string{"a": "b", "c": ""}), "myflag", "doc")
	cmd := &Command{flagSet: set}
	f := &StringMapFlag{Name: "myflag"}
	require.Equal(t, map[string]string{"a": "b", "c": ""}, f.Get(cmd))
}

func TestStringMapFlagApply_Error(t *testing.T) {
	fl := StringMapFlag{Name: "goat"}
	set := flag.NewFlagSet("test", 0)
	_ = fl.Apply(set)

	err := set.Parse([]string{"--goat", "aaa", "bbb="})
	if err == nil {
		t.Errorf("expected error, but got none")
	}
}
