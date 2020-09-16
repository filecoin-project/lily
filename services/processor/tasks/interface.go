package tasks

import "github.com/gocraft/work"

type ProcessTask interface {
	Log(job *work.Job, next work.NextMiddlewareFunc) error
	ParseArgs(job *work.Job) error
	Task(job *work.Job) error
}
