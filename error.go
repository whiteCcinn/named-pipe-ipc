package named_pipe_ipc

type AlreadyExistButNotNamedPipe struct {
}

func (e AlreadyExistButNotNamedPipe) Error() string {
	return "Already exist but which not named pipe"
}

type NotDirectory struct {
}

func (e NotDirectory) Error() string {
	return "It is not a directory, because it has to be a directory"
}
