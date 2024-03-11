package concurrency

import (
	"context"
	"fmt"
)

type TaskID uint32
type TaskType string
type TaskMetadata map[string]interface{}

type ExecutionFn func(ctx context.Context, args interface{}) (interface{}, error)

type TaskDescriptor struct {
	Metadata TaskMetadata
	TaskType TaskType
	ID       TaskID
}

type Result struct {
	Value      interface{}
	Err        error
	Descriptor TaskDescriptor
}

type Task struct {
	Args       interface{}
	ExecFn     ExecutionFn
	Descriptor TaskDescriptor
}

// execute the task provided function with arguments.
func (t Task) execute(ctx context.Context) Result {
	// this is pre-cautionary check, noticed failure couple of times because of fake task.
	// need to debug more on this.
	if t.Args == nil || t.ExecFn == nil {
		return Result{Err: fmt.Errorf("fake task")}
	}

	value, err := t.ExecFn(ctx, t.Args)
	if err != nil {
		return Result{
			Err:        err,
			Descriptor: t.Descriptor,
		}
	}

	return Result{
		Value:      value,
		Descriptor: t.Descriptor,
	}
}
