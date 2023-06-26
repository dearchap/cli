package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"
)

const (
	contextContextKey = contextKey("cli.context")
)

type contextKey string

// Context is a type that is passed through to
// each Handler action in a cli application. Context
// can be used to retrieve context-specific args and
// parsed command-line options.
type Context struct {
	context.Context
	Command       *Command
	shellComplete bool
	flagSet       *flag.FlagSet
	parent        *Context
}

// NewContext creates a new context. For use in when invoking a Command action.
func NewContext(cmd *Command, set *flag.FlagSet, parent *Context) *Context {
	cCtx := &Context{
		Command: cmd,
		flagSet: set,
		parent:  parent,
	}

	if parent != nil {
		cCtx.Context = parent.Context
		cCtx.shellComplete = parent.shellComplete

		if parent.flagSet == nil {
			parent.flagSet = &flag.FlagSet{}
		}
	}

	if cCtx.Command == nil {
		cCtx.Command = &Command{}
	}

	if cCtx.Context == nil {
		cCtx.Context = context.Background()
	}

	return cCtx
}

// NumFlags returns the number of flags set
func (cCtx *Context) NumFlags() int {
	return cCtx.flagSet.NFlag()
}

// Set sets a context flag to a value.
func (cCtx *Context) Set(name, value string) error {
	if fs := cCtx.lookupFlagSet(name); fs != nil {
		return fs.Set(name, value)
	}

	return fmt.Errorf("no such flag -%s", name)
}

// IsSet determines if the flag was actually set
func (cCtx *Context) IsSet(name string) bool {
	if fs := cCtx.lookupFlagSet(name); fs != nil {
		isSet := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == name {
				isSet = true
			}
		})
		if isSet {
			return true
		}

		f := cCtx.lookupFlag(name)
		if f == nil {
			return false
		}

		return f.IsSet()
	}

	return false
}

// LocalFlagNames returns a slice of flag names used in this context.
func (cCtx *Context) LocalFlagNames() []string {
	var names []string
	cCtx.flagSet.Visit(makeFlagNameVisitor(&names))
	// Check the flags which have been set via env or file
	if cCtx.Command != nil && cCtx.Command.Flags != nil {
		for _, f := range cCtx.Command.Flags {
			if f.IsSet() {
				names = append(names, f.Names()...)
			}
		}
	}

	// Sort out the duplicates since flag could be set via multiple
	// paths
	m := map[string]struct{}{}
	var unames []string
	for _, name := range names {
		if _, ok := m[name]; !ok {
			m[name] = struct{}{}
			unames = append(unames, name)
		}
	}

	return unames
}

// FlagNames returns a slice of flag names used by the this context and all of
// its parent contexts.
func (cCtx *Context) FlagNames() []string {
	var names []string
	for _, pCtx := range cCtx.Lineage() {
		names = append(names, pCtx.LocalFlagNames()...)
	}
	return names
}

// Lineage returns *this* context and all of its ancestor contexts in order from
// child to parent
func (cCtx *Context) Lineage() []*Context {
	var lineage []*Context

	for cur := cCtx; cur != nil; cur = cur.parent {
		lineage = append(lineage, cur)
	}

	return lineage
}

// Count returns the num of occurrences of this flag
func (cCtx *Context) Count(name string) int {
	if fs := cCtx.lookupFlagSet(name); fs != nil {
		if cf, ok := fs.Lookup(name).Value.(Countable); ok {
			return cf.Count()
		}
	}
	return 0
}

// Value returns the value of the flag corresponding to `name`
func (cCtx *Context) Value(name string) interface{} {
	if fs := cCtx.lookupFlagSet(name); fs != nil {
		return fs.Lookup(name).Value.(flag.Getter).Get()
	}
	return nil
}

// Args returns the command line arguments associated with the context.
func (cCtx *Context) Args() Args {
	ret := args(cCtx.flagSet.Args())
	return &ret
}

// NArg returns the number of the command line arguments.
func (cCtx *Context) NArg() int {
	return cCtx.Args().Len()
}

func (cCtx *Context) lookupFlag(name string) Flag {
	for _, c := range cCtx.Lineage() {
		if c.Command == nil {
			continue
		}

		for _, f := range c.Command.Flags {
			for _, n := range f.Names() {
				if n == name {
					return f
				}
			}
		}
	}

	return nil
}

func (cCtx *Context) lookupFlagSet(name string) *flag.FlagSet {
	for _, c := range cCtx.Lineage() {
		if c.flagSet == nil {
			continue
		}
		if f := c.flagSet.Lookup(name); f != nil {
			return c.flagSet
		}
	}
	cCtx.onInvalidFlag(name)
	return nil
}

func (cCtx *Context) checkRequiredFlags(flags []Flag) requiredFlagsErr {
	var missingFlags []string
	for _, f := range flags {
		if rf, ok := f.(RequiredFlag); ok && rf.IsRequired() {
			var flagPresent bool
			var flagName string

			for _, key := range f.Names() {
				flagName = key

				if cCtx.IsSet(strings.TrimSpace(key)) {
					flagPresent = true
				}
			}

			if !flagPresent && flagName != "" {
				missingFlags = append(missingFlags, flagName)
			}
		}
	}

	if len(missingFlags) != 0 {
		return &errRequiredFlags{missingFlags: missingFlags}
	}

	return nil
}

func (cCtx *Context) onInvalidFlag(name string) {
	for cCtx != nil {
		if cCtx.Command != nil && cCtx.Command.InvalidFlagAccessHandler != nil {
			cCtx.Command.InvalidFlagAccessHandler(cCtx, name)
			break
		}
		cCtx = cCtx.parent
	}
}

func makeFlagNameVisitor(names *[]string) func(*flag.Flag) {
	return func(f *flag.Flag) {
		nameParts := strings.Split(f.Name, ",")
		name := strings.TrimSpace(nameParts[0])

		for _, part := range nameParts {
			part = strings.TrimSpace(part)
			if len(part) > len(name) {
				name = part
			}
		}

		if name != "" {
			*names = append(*names, name)
		}
	}
}
