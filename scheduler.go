package tinyscheduler

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// +-------------------------+
// |       main()            |
// | +---------------------+ |
// | |   TaskScheduler     | |
// | +---------------------+ |
// |         | ^             |
// |         | | Start       |
// |         v |             |
// | +---------------------+ |          +------------------+
// | |   TaskScheduler     | |          |    time.Ticker   |
// | |     .Start()        | | -------> |for each interval |
// | +---------------------+ |          +------------------+
// +-------------------------+                  |
//            |                                 | Ticks
//            |                                 v
//            |            +-----------------------------+
//            |            |    Goroutine per interval   |
//            |            | +-------------------------+ |
//            |            | |   Range over tasks map  | |
//            |            +-> | (for each interval)   | |
//            |              | +-------------------------|
//            |              +-----------------------------+
//            |                        |
//            |                        | Executes
//            |                        v
//            |              +-----------------------------+
//            +------------> |       Task execution        |
//                           |       (Handler func)        |
//                           +-----------------------------+
//
// main() function: Initializes and controls the TaskScheduler.
// TaskScheduler: Manages the scheduling and execution of tasks.
// TaskScheduler.Start(): Starts the scheduling process, creating time.Ticker for each interval.
// time.Ticker: Triggers events (ticks) at specified intervals.
// Goroutine per interval: Each interval has a separate goroutine that executes tasks for that interval.
// Range over tasks map: Iterates over tasks scheduled for execution at the current interval.
// Task execution: Executes the associated handler function of each task.

type Task struct {
	//ID is a unique identifier for the task
	//If ID is not set, a random ID is generated
	ID string
	//Handler is a function that executes the task
	//If Handler is not set, the task is ignored
	//If Handler is set, the task is executed
	//If Handler is set to nil, the task is ignored
	Handler func() error

	//ErrorHandler is a function that handles errors
	//If ErrorHandler is not set, the error is logged
	//If ErrorHandler is set, the error is passed to the function
	//If ErrorHandler is set to nil, the error is ignored
	ErrorHandler func(error)

	//Timeout is the time after which the task is considered to have timed out
	Timeout time.Duration
}

// Logger is an interface for logging
// It is used to mock log.Logger in tests
type Logger interface {
	Printf(format string, v ...interface{})
}

// TimeManager is an interface for time management
// It is used to mock time.Ticker in tests
type TimeManager interface {
	NewTicker(duration time.Duration) *time.Ticker
}

// NewTask creates a new task
// If ID is not set, a random ID is generated
// If Handler is not set, the task is ignored
func NewTask(handler func() error) Task {
	return Task{
		ID:      uuid.New().String(),
		Handler: handler,
	}
}

// HandleError sets the ErrorHandler for the task. The ErrorHandler is a function that handles errors occurring during task execution.
// - If ErrorHandler is not set (nil), any error that occurs is logged.
// - If ErrorHandler is provided, it's called with the error that occurred during task execution.
// - If the ErrorHandler itself panics, the panic is recovered and logged.
// - The return value of the ErrorHandler is not used; hence, if it returns an error, this error is not further processed.
func (t *Task) HandleError(errorHandler func(err error)) {
	t.ErrorHandler = errorHandler
}

// TaskScheduler manages the scheduling and execution of tasks
// It has a tasks map, quit channel, logger, done channel and time manager
// The tasks map is used to store tasks for each interval
// The quit channel is used to signal the scheduler to stop
type TaskScheduler struct {
	tasks       map[time.Duration][]Task
	quit        chan struct{}
	logger      Logger
	wg          sync.WaitGroup
	done        chan struct{}
	timeManager TimeManager
	timeout     time.Duration
}

// NewTaskScheduler creates a new task scheduler
// It initializes the tasks map, quit channel, logger, done channel and time manager
// The tasks map is used to store tasks for each interval
// The quit channel is used to signal the scheduler to stop
// The logger is used to log messages
// The done channel is used to signal the scheduler has stopped
// The time manager is used to create time.Ticker
func NewTaskScheduler() *TaskScheduler {
	logger := log.New(os.Stdout, "TinyScheduler: ", log.LstdFlags)
	logger.SetFlags(log.LstdFlags)
	return &TaskScheduler{
		tasks:       make(map[time.Duration][]Task),
		quit:        make(chan struct{}),
		logger:      logger,
		done:        make(chan struct{}),
		timeManager: &RealTimeManager{},
	}
}

// AddTask adds a task to the scheduler
// interval is the time between executions
// task is the function to execute
// The task is executed asynchronously in a new goroutine
func (s *TaskScheduler) AddTask(interval time.Duration, task Task) {
	s.tasks[interval] = append(s.tasks[interval], task)
}

// Start begins the task execution in the scheduler
// It starts new goroutines for each task interval
func (s *TaskScheduler) Start() {
	for interval, tasks := range s.tasks { // Iterate over each interval and its associated tasks
		ticker := s.timeManager.NewTicker(interval) // Create a new ticker for the interval
		s.wg.Add(1)
		go func(ticker *time.Ticker, tasks []Task) { // Start a new goroutine for each interval
			defer s.wg.Done()   // Decrement the WaitGroup counter after the goroutine is done
			defer ticker.Stop() // Stop the ticker after the goroutine is done
			for {
				select {
				case <-ticker.C: // When the ticker ticks...
					for _, t := range tasks { // Iterate over each task for the interval
						s.wg.Add(1)   // Increment the WaitGroup counter before starting a new task
						taskCopy := t // Avoid data race by copying the task
						go s.executeTask(taskCopy)
					}
				case <-s.quit:
					return
				}
			}
		}(ticker, tasks)
	}
	// Wait for all tasks to complete before returning
	s.wg.Wait()

	// Block until done is closed
	<-s.done
}

func (s *TaskScheduler) executeTask(task Task) {
	defer func() {
		s.wg.Done()                   // Decrement the WaitGroup counter after the task is done
		if r := recover(); r != nil { // Recover from any panics in the task
			s.logger.Printf("ERROR: Task %s - %v", task.ID, r)
		}
	}()

	s.logger.Printf("INFO: Task started id: %s", task.ID) // Log task start
	if err := task.Handler(); err != nil {                // Execute the task; if it returns an error...
		task.ErrorHandler(err)                               // Handle the error
		s.logger.Printf("ERROR: Task %s - %v", task.ID, err) // Log the error
	} else {
		s.logger.Printf("INFO: Task completed successfully id: %s", task.ID) // Log task completion
	}
}

// Shutdown stops the scheduler
func (s *TaskScheduler) Shutdown() {
	s.logger.Printf("Scheduler shutting down...")

	// Stop the ticker
	s.quit <- struct{}{}

	//s.wg.Done()
	// Wait for all running tasks to finish
	// why do we need this?
	// because we want to wait for all tasks to finish
	s.wg.Wait()

	s.logger.Printf("Scheduler shutdown completed")

	// send done signal
	// this will unblock the Start() method and return
	// why send a struct{}{}?
	// because we want to signal that the scheduler is done
	s.done <- struct{}{}
}

// ListenForGracefullyShutdown listens for SIGTERM, SIGQUIT, SIGINT and calls Shutdown() when received
// This allows the scheduler to shut down gracefully
// This is useful for running the scheduler in a container
// SIGTERM is sent by default when running `docker stop`
// SIGQUIT is sent by default when running `docker stop -t 0`
// SIGINT is sent by default when running `docker stop` or `CTRL+C`
func (s *TaskScheduler) ListenForGracefullyShutdown() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	go func() {
		<-sigc
		s.logger.Printf("Scheduler shutting down gracefully...")
		s.Shutdown()
	}()
}
