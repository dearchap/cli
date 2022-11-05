package cli

import (
	"flag"
	"strconv"
)

// -- uint64 Value
type uint64Value uint64

func (i uint64Value) Create(val uint64, p *uint64) flag.Value {
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		return err
	}
	*i = uint64Value(v)
	return err
}

func (i *uint64Value) Get() any { return uint64(*i) }

func (i *uint64Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

type Uint64Flag = flagImpl[uint64, uint64Value]

// Int64 looks up the value of a local Int64Flag, returns
// 0 if not found
func (cCtx *Context) Uint64(name string) uint64 {
	if v, ok := cCtx.Value(name).(uint64); ok {
		return v
	}
	return 0
}
