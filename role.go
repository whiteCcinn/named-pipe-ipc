package named_pipe_ipc

type RoleType int

const (
	C RoleType = iota + 1
	S
)

func (r RoleType) String() (s string) {
	switch r {
	case C:
		s = "This is Client Role, Pipe file normal read and write"
	case S:
		s = "This is Server Role, Pipe file reverse"
	default:
		s = "Unknown RoleType"
	}
	return
}
