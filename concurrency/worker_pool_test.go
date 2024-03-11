package concurrency

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	tasksCount  = 703
	workerCount = 4
)

func Test_workerReadLoop(t *testing.T) {
	workerID := 0
	workerName := "test-worker"

	stopChan := make(chan bool, 1)
	inputReadyChan := make(chan bool, 1)
	nextInputReadyChan := make(chan bool, 1)

	task := &Task{
		Descriptor: TaskDescriptor{
			ID:       TaskID(1),
			TaskType: "Test Type",
			Metadata: nil,
		},
		ExecFn: execFn,
		Args:   1,
	}
	taskChannel := make(chan Task, 1)
	taskChannel <- *task
	t0 := time.Now()
	inputReadyChan <- true
	actualTask, stop, err := workerReadLoop(0, workerName,
		taskChannel, stopChan, inputReadyChan, nextInputReadyChan, t0, time.Nanosecond)
	assert.NotNil(t, actualTask)
	assert.Nil(t, err)
	assert.False(t, stop)
	assert.True(t, <-nextInputReadyChan)

	// testing stop channel flow.
	t0 = time.Now()
	stopChan <- true
	actualTask, stop, err = workerReadLoop(workerID, "test-worker",
		taskChannel, stopChan, inputReadyChan, nextInputReadyChan, t0, time.Nanosecond)
	assert.Nil(t, err)
	assert.Nil(t, actualTask)
	assert.True(t, stop)

	// testing timeout channel flow.
	t0 = time.Now()
	actualTask, stop, err = workerReadLoop(0, "test-worker",
		taskChannel, stopChan, inputReadyChan, nextInputReadyChan, t0, time.Microsecond*11)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("%s, worker %d: chan read loop timeout", workerName, workerID), err)
	assert.Nil(t, actualTask)
	assert.True(t, stop)

	close(stopChan)
	close(taskChannel)
	close(inputReadyChan)
	close(nextInputReadyChan)
}

func Test_workerWriteLoop(t *testing.T) {
	workerID := 0
	workerName := "test-worker"

	stopChan := make(chan bool, 1)
	outputReadyChan := make(chan bool, 1)
	nextOutputReadyChan := make(chan bool, 1)

	resultChannel := make(chan Result, 1)
	result := Result{}
	t0 := time.Now()
	outputReadyChan <- true
	stop, err := workerWriteLoop(0, workerName,
		result, resultChannel,
		stopChan, outputReadyChan, nextOutputReadyChan, t0, time.Nanosecond)
	assert.Nil(t, err)
	assert.False(t, stop)
	assert.NotNil(t, <-resultChannel)

	// testing stop channel flow.
	t0 = time.Now()
	stopChan <- true
	stop, err = workerWriteLoop(0, workerName,
		result, resultChannel,
		stopChan, outputReadyChan, nextOutputReadyChan, t0, time.Nanosecond)
	assert.Nil(t, err)
	assert.True(t, stop)

	// testing timeout channel flow.
	t0 = time.Now()
	stop, err = workerWriteLoop(0, workerName,
		result, resultChannel,
		stopChan, outputReadyChan, nextOutputReadyChan, t0, time.Nanosecond)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("%s, worker %d: chan write loop timeout", workerName, workerID), err)
	assert.True(t, stop)

	close(stopChan)
	close(resultChannel)
	close(outputReadyChan)
	close(nextOutputReadyChan)
}

// Test_WorkerPool, this is the happy path for the worker pool pattern
// all submitted tasks are executed and the result channel has the results for all tasks
// Also, the done channel has the true value showing the execution is finished for all tasks
func Test_WorkerPool_AllTaskExecuted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := New(workerCount, tasksCount, "Test_WorkerPool_AllTaskExecuted")
	tasksArray := getTestTasks()
	wp.GenerateFrom(tasksArray)
	go wp.Run(ctx)

	executedTaskResultCount := 0
	for result := range wp.Results {
		assert.NotNil(t, result.Descriptor) // asserting that the Descriptor is not nil

		i := result.Descriptor.ID
		actualVal := result.Value.(int)
		executedTaskResultCount++ // incrementing the result counter, it should be equal in the end with total tasks

		assert.NotNil(t, actualVal)
		assert.Equal(t, int(i), executedTaskResultCount) // to test that order is preserved
		assert.True(t, result.Err == nil)
		assert.Equal(t, int(i-1)*2, actualVal)
	}
	assert.Equal(t, len(tasksArray), executedTaskResultCount)
}

// TestWorkerPool_TimeOut, this is the test to verify the timeout option for the worker set at the context level
func TestWorkerPool_TimeOut(t *testing.T) {
	// setting the timeout time as 10 Nanosecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := NewWithLoopTimeOut(workerCount, tasksCount, "TestWorkerPool_TimeOut", time.Nanosecond*10)
	close(wp.Tasks)
	go wp.Run(ctx)

	executedTaskResultCount := 0
	for range wp.Results {
		executedTaskResultCount++
	}
	assert.Equal(t, 0, executedTaskResultCount)
}

// Test_WorkerPool_Cancel, this is the test to verify the cancel option for the worker set at the context level
func Test_WorkerPool_Cancel(t *testing.T) {
	wp := New(DefaultWorkerCount, tasksCount, "Test_WorkerPool_Cancel")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tasksArray := getTestTasks()
	wp.GenerateFrom(tasksArray)
	go wp.Run(ctx)

	executedTaskResultCount := 0
	breakExecutionCount := 38
	for result := range wp.Results {
		if executedTaskResultCount == breakExecutionCount {
			wp.StopWorkers()
		}

		assert.True(t, result.Err == nil)
		assert.NotNil(t, result.Value)
		assert.NotNil(t, result.Descriptor)
		executedTaskResultCount++
	}
	println(executedTaskResultCount)
	assert.GreaterOrEqualf(t, executedTaskResultCount, breakExecutionCount, "executed task result count should be greater or equal to breakExecutionCount")
}

func getTestTasks() []Task {
	tasks := make([]Task, tasksCount)
	for i := 0; i < tasksCount; i++ {
		tasks[i] = Task{
			Descriptor: TaskDescriptor{
				ID:       TaskID(i + 1),
				TaskType: "Test Type",
				Metadata: nil,
			},
			ExecFn: execFn,
			Args:   i,
		}
	}
	return tasks
}
