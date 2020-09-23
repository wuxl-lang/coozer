package coozer

const (
	_ = 1 << iota 	// 1 << 0
	_				// 1 << 1
	set				// 1 << 2
	del				// 1 << 3
)

// Event describes changes of specific path
type Event struct {
	Rev  int64
	Path string
	Body []byte
	Flag int32
}

// IsSet decides if a path is updated
func (e *Event) IsSet() bool {
	return e.Flag & set > 0
}

// IsDel decides if a path is deleted
func (e *Event) IsDel() bool {
	return e.Flag & del > 0
}
