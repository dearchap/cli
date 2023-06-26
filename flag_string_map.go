package cli

type StringMap = MapBase[string, StringConfig, stringValue]
type StringMapFlag = FlagBase[map[string]string, StringConfig, StringMap]

var NewStringMap = NewMapBase[string, StringConfig, stringValue]

// StringMap looks up the value of a local StringMapFlag, returns
// nil if not found
func (cCtx *Context) StringMap(name string) map[string]string {
	if v, ok := cCtx.Value(name).(map[string]string); ok {
		return v
	}
	return nil
}
