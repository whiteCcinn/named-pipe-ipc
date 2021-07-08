package named_pipe_ipc

import "fmt"

const (
	AlreadyExistButNotNamedPipeMessage = "Already exist but which not named pipe"
	NotDirectoryMessage                = "It is not a directory, because it has to be a directory"
	NoMessageMessage                   = "Not receive message"
	MessageNotLegalMessage             = "Message is not legal"
	NoPipeExistMessage                 = "No pipe exist"
	PipeClosedMessage                  = "pipe closed"
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

type MessageNotLegal struct {
}

func (e MessageNotLegal) Error() string {
	return MessageNotLegalMessage
}

type NoMessage struct {
}

func (e NoMessage) Error() string {
	return NoMessageMessage
}

type NoPipeExist struct {
}

func (e NoPipeExist) Error() string {
	return NoPipeExistMessage
}

type Closed struct {
}

func (e Closed) Error() string {
	return PipeClosedMessage
}

type HybridError struct {
	EA error
	EB error
}

func (e HybridError) Error() string {
	return fmt.Sprintf("EA: %v, EB: %v", e.EA, e.EB)
}
