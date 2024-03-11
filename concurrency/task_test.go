package concurrency

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	errDefault = errors.New("wrong argument type")
	descriptor = TaskDescriptor{
		ID:       TaskID(1),
		TaskType: TaskType("anyType"),
		Metadata: TaskMetadata{
			"foo": "foo",
			"bar": "bar",
		},
	}
	execFn = func(ctx context.Context, args interface{}) (interface{}, error) {
		argVal, ok := args.(int)
		if !ok {
			return nil, errDefault
		}

		return argVal * 2, nil
	}
)

func Test_TaskExecute(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name     string
		task     Task
		expected Result
	}{
		{
			name: "Task execution success",
			task: Task{
				Descriptor: descriptor,
				ExecFn:     execFn,
				Args:       10,
			},
			expected: Result{
				Value:      20,
				Descriptor: descriptor,
			},
		},
		{
			name: "Task execution failure",
			task: Task{
				Descriptor: descriptor,
				ExecFn:     execFn,
				Args:       "10",
			},
			expected: Result{
				Err:        errDefault,
				Descriptor: descriptor,
			},
		},
		{
			name: "Fake task execution handling",
			task: Task{},
			expected: Result{
				Err: fmt.Errorf("fake task"),
			},
		},
		{
			name: "Missing task arguments",
			task: Task{},
			expected: Result{
				Err: fmt.Errorf("fake task"),
			},
		},
		{
			name: "Missing task execution function",
			task: Task{},
			expected: Result{
				Err: fmt.Errorf("fake task"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.execute(ctx)
			assert.Equal(t, got.Err, tt.expected.Err)
			assert.Equal(t, got, tt.expected)
		})
	}
}
