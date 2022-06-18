package main

type botError struct {
	msg string
}

func NewBotError(msg string) *botError {
	return &botError{msg}
}

func (e *botError) Error() string {
	return e.msg
}
