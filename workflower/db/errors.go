package db

import "errors"

var (
	ErrAlreadyRunning  = errors.New("already running")
	ErrAlreadyFinished = errors.New("already finished")
	ErrIsNotRunning    = errors.New("not running")
)
