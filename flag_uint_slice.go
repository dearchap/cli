package cli

type UintSlice = SliceBase[uint, IntegerConfig, uintValue]
type UintSliceFlag = FlagBase[[]uint, IntegerConfig, UintSlice]

var NewUintSlice = NewSliceBase[uint, IntegerConfig, uintValue]

// UintSlice looks up the value of a local UintSliceFlag, returns
// nil if not found
func (cCtx *Context) UintSlice(name string) []uint {
	if v, ok := cCtx.Value(name).([]uint); ok {
		return v
	}
	return nil
}
