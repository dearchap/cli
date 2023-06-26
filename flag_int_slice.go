package cli

type IntSlice = SliceBase[int64, IntegerConfig, intValue]
type IntSliceFlag = FlagBase[[]int64, IntegerConfig, IntSlice]

var NewIntSlice = NewSliceBase[int64, IntegerConfig, intValue]

// IntSlice looks up the value of a local IntSliceFlag, returns
// nil if not found
func (cCtx *Context) IntSlice(name string) []int64 {
	if v, ok := cCtx.Value(name).([]int64); ok {
		return v
	}

	return nil
}
