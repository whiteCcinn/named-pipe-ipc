package named_pipe_ipc

const (
	AlreadyExistButNotNamedPipeMessage = "Already exist but which not named pipe"
	NotDirectoryMessage                = "It is not a directory, because it has to be a directory"
	NoMessageMessage                   = "Not receive message"
)

type AlreadyExistButNotNamedPipe struct {
}

func (e AlreadyExistButNotNamedPipe) Error() string {
	return AlreadyExistButNotNamedPipeMessage
}

type NotDirectory struct {
}

func (e NotDirectory) Error() string {
	return NotDirectoryMessage
}

type NoMessage struct {
}

func (e NoMessage) Error() string {
	return NoMessageMessage
}
