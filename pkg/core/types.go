package core

// ChangeType denotes type of state change
type ChangeType uint8

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeAdd
	ChangeTypeRemove
	ChangeTypeModify
)

func (c ChangeType) String() string {
	switch c {
	case ChangeTypeUnknown:
		return "unknown"
	case ChangeTypeAdd:
		return "add"
	case ChangeTypeRemove:
		return "remove"
	case ChangeTypeModify:
		return "modify"
	}
	panic("unreachable")
}
