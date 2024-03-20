package concurrency

import (
	"context"
	"fmt"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
	"runtime"
	"sync"
	"time"
)

const LoopTimeout = time.Second * 20
const SleepTime = time.Microsecond * 5

// DefaultWorkerCount default workers count, depend on runtime cpu value
var DefaultWorkerCount = runtime.NumCPU()

func workerReadLoop(
	workerID int, workerName string,                        // worker metadata for tracking/debugging purpose
	tasks chan Task,                                        // input args
	stopChan, inputReadyChan, nextInputReadyChan chan bool, // synchronization channels args
	initialTime time.Time, timeOut time.Duration) (task *Task, stop bool, err error) {
	for {
		select {
		case <-stopChan:
			return nil, true, nil
		case <-inputReadyChan:
			// 	ready to read
		default:
			if time.Since(initialTime) > timeOut {
				return nil, true, fmt.Errorf("%s, worker %d: chan read loop timeout", workerName, workerID)
			}
			time.Sleep(SleepTime)
			continue
		}

		task, ok := <-tasks

		nextInputReadyChan <- true // this has to be done before closing the worker read loop, to close the other workers
		if !ok {
			// This is an empty value, no task left on the channel
			return nil, true, nil
		}

		return &task, false, nil
	}
}

func workerWriteLoop(
	workerID int, workerName string,                          // worker metadata for tracking/debugging purpose
	result Result, outputChan chan Result,                    // results args
	stopChan, outputReadyChan, nextOutputReadyChan chan bool, // synchronization channels args
	initialTime time.Time, timeOut time.Duration) (stop bool, err error) {
	for {
		select {
		case <-stopChan:
			return true, nil
		case <-outputReadyChan:
			// 	ready to write
		default:
			if time.Since(initialTime) > timeOut {
				return true, fmt.Errorf("%s, worker %d: chan write loop timeout", workerName, workerID)
			}
			time.Sleep(SleepTime)
			continue
		}
		outputChan <- result
		nextOutputReadyChan <- true
		return false, nil
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup,     // context and wait group args
	workerID int, workerName string,                     // worker metadata for tracking/debugging purpose
	tasks chan Task, results chan Result,                // input task and output result channel args
	stopChan, inputReadyChan, outputReadyChan chan bool, // current worker synchronization channels args
	nextInputReadyChan, nextOutputReadyChan chan bool,   // next worker synchronization channels args
	timeOut time.Duration) {
	defer wg.Done()

	span, _ := tracing.StartSpanFromContext(ctx, fmt.Sprintf("%s worker %d", workerName, workerID))
	defer span.End()

	t0 := time.Now()
	for {
		task, stop, err := workerReadLoop(workerID, workerName,
			tasks, stopChan, inputReadyChan, nextInputReadyChan, t0, timeOut)
		if err != nil {
			log.Get(ctx).Error(err.Error())
			break
		}
		if stop {
			break
		}

		result := task.execute(ctx) // executing the actual compute function

		stop, err = workerWriteLoop(workerID, workerName,
			result, results, stopChan, outputReadyChan, nextOutputReadyChan, t0, timeOut)
		if err != nil {
			log.Get(ctx).Error(err.Error())
			break
		}
		if stop {
			break
		}
	}
}

type WorkerPool struct {
	Tasks                    chan Task   // input tasks to work parallel following order
	Results                  chan Result // output result channel, results will be in order as per input task order
	WorkerPoolName           string
	WorkerStopChannel        []chan bool   // worker stop channel
	WorkerInputReadyChannel  []chan bool   // worker input ready channel
	WorkerOutputReadyChannel []chan bool   // worker output ready channel
	LoopTimeOut              time.Duration // read loop timeout limit, default 2 seconds
	WorkersCount             int
	TasksCount               int
}

func NewWithLoopTimeOut(workerCount, tasksCount int, workerPoolName string, loopTimeOut time.Duration) WorkerPool {
	return WorkerPool{
		WorkersCount:             workerCount,
		TasksCount:               tasksCount,
		WorkerStopChannel:        make([]chan bool, workerCount),
		WorkerInputReadyChannel:  make([]chan bool, workerCount),
		WorkerOutputReadyChannel: make([]chan bool, workerCount),
		Tasks:                    make(chan Task, tasksCount),
		Results:                  make(chan Result, tasksCount),
		WorkerPoolName:           workerPoolName,
		LoopTimeOut:              loopTimeOut,
	}
}

func New(workerCount, tasksCount int, workerPoolName string) WorkerPool {
	return NewWithLoopTimeOut(workerCount, tasksCount, workerPoolName, LoopTimeout)
}

func (wp WorkerPool) Run(ctx context.Context) {
	wg := sync.WaitGroup{}

	// creating workers synchronization channels
	for i := 0; i < wp.WorkersCount; i++ {
		wp.WorkerStopChannel[i] = make(chan bool, 1)
		wp.WorkerInputReadyChannel[i] = make(chan bool, 1)
		wp.WorkerOutputReadyChannel[i] = make(chan bool, 1)
	}

	// loop to start the workers for execution
	for i := 0; i < wp.WorkersCount; i++ {
		wg.Add(1)

		// required for next work synchronization channel
		nextWorkerIdx := addMod(i, 1, wp.WorkersCount)

		// fan out worker goroutines
		// reading from tasks channel and
		// pushing the calculation into results channel
		go worker(ctx, &wg,
			i, wp.WorkerPoolName,
			wp.Tasks, wp.Results,
			wp.WorkerStopChannel[i], wp.WorkerInputReadyChannel[i], wp.WorkerOutputReadyChannel[i],
			wp.WorkerInputReadyChannel[nextWorkerIdx], wp.WorkerOutputReadyChannel[nextWorkerIdx],
			wp.LoopTimeOut)
	}

	// start the workers by marking first worker channels as ready
	wp.WorkerInputReadyChannel[0] <- true
	wp.WorkerOutputReadyChannel[0] <- true

	// wait for the workers to finish the work
	wg.Wait()

	// collect the last results from the workers result channel and close the channel
	// NOTE: if the workers finished the execution of all the tasks, in that case StopWorker channel will not be closed.
	// As per the Google group discussion(https://groups.google.com/g/golang-nuts/c/pZwdYRGxCIk/m/qpbHxRRPJdUJ?pli=1)
	// Note that it is only necessary to close a channel if the receiver is looking for a close. Closing the channel is a control signal on the channel indicating that no more data follows.
	close(wp.Results) // closing the buffered result channel
}

func (wp WorkerPool) StopWorkers() {
	for i := 0; i < wp.WorkersCount; i++ {
		wp.WorkerStopChannel[i] <- true
		close(wp.WorkerStopChannel[i])
	}
}

func (wp WorkerPool) GenerateFrom(bulkTasks []Task) {
	for i := range bulkTasks {
		wp.Tasks <- bulkTasks[i]
	}
	close(wp.Tasks)
}

// addMod: Modular addition of x and y. x and y must be non-negative. mod must be positive.
// This function is used to assign the next worker input and output ready synchronization channels.
//
// for example, with 4 workers, here it looks like
//
//	worker 0 with worker 1 next input and output ready synchronization, (0+1) % 4 (x = worker idx, y = 1) = 1
//	worker 1 with worker 2 next input and output ready synchronization, (1+1) % 4 (x = worker idx, y = 1) = 2
//	worker 2 with worker 3 next input and output ready synchronization, (2+1) % 4 (x = worker idx, y = 1) = 3
//	worker 3 with worker 0 next input and output ready synchronization, (3+1) % 4 (x = worker idx, y = 1) = 0
func addMod(x, y, mod int) int {
	return (x + y) % mod
}
