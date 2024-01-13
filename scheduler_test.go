package tinyscheduler

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func TestTaskExecution(t *testing.T) {

	// Arrange
	taskExecuted := false
	taskCallCount := 0
	task := NewTask(func() error {
		taskExecuted = true
		taskCallCount++
		fmt.Println("Hello World")
		return nil
	})

	mockTimeManager := NewMockTimeManager()

	sc := TaskScheduler{
		tasks:       make(map[time.Duration][]Task),
		quit:        make(chan struct{}),
		logger:      log.New(os.Stdout, "TINYSCHEDULER: ", log.LstdFlags),
		wg:          sync.WaitGroup{},
		done:        make(chan struct{}),
		timeManager: mockTimeManager,
		timeout:     time.Second * 1,
	}
	sc.AddTask(time.Second*1, task)

	// Act
	go sc.Start()
	mockTimeManager.C <- time.Now()
	mockTimeManager.C <- time.Now()
	mockTimeManager.C <- time.Now()

	time.Sleep(100 * time.Millisecond)

	// Assert
	if !taskExecuted {
		t.Errorf("Expected task to be executed, but it was not.")
	}

	if taskCallCount != 3 {
		t.Errorf("Expected task to be executed 3 times, but it was executed %d times.", taskCallCount)
	}
}

func TestMultipleTasksExecution(t *testing.T) {
	mockTM := NewMockTimeManager()
	scheduler := TaskScheduler{
		tasks:       make(map[time.Duration][]Task),
		quit:        make(chan struct{}),
		logger:      log.New(os.Stdout, "TINYSCHEDULER: ", log.LstdFlags),
		wg:          sync.WaitGroup{},
		done:        make(chan struct{}),
		timeManager: mockTM,
		timeout:     time.Second * 1,
	}

	executedTasks := make(map[string]bool)
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		scheduler.AddTask(time.Second, NewTask(func() error {
			executedTasks[taskID] = true
			return nil
		}))
	}

	go scheduler.Start()

	// Simulate 3 ticks
	mockTM.C <- time.Now()

	time.Sleep(100 * time.Millisecond)

	if len(executedTasks) != 3 {
		t.Errorf("Expected 3 tasks to be executed, got %d", len(executedTasks))
	}

	scheduler.Shutdown()
}

func TestTaskExecutionFailure(t *testing.T) {
	mockTM := NewMockTimeManager()
	scheduler := TaskScheduler{
		tasks:       make(map[time.Duration][]Task),
		quit:        make(chan struct{}),
		logger:      log.New(os.Stdout, "TINYSCHEDULER: ", log.LstdFlags),
		wg:          sync.WaitGroup{},
		done:        make(chan struct{}),
		timeManager: mockTM,
		timeout:     time.Second * 1,
	}

	errorHandled := false
	task := NewTask(func() error {
		return fmt.Errorf("error executing task")
	})
	task.HandleError(func(err error) {
		if err != nil {
			errorHandled = true
		}
	})

	scheduler.AddTask(time.Second, task)

	go scheduler.Start()
	mockTM.C <- time.Now()

	time.Sleep(100 * time.Millisecond)

	if !errorHandled {
		t.Errorf("Error handler was not invoked on task failure")
	}

	scheduler.Shutdown()
}

func TestSchedulerShutdown(t *testing.T) {
	mockTM := NewMockTimeManager()
	scheduler := TaskScheduler{
		tasks:       make(map[time.Duration][]Task),
		quit:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		done:        make(chan struct{}),
		timeManager: mockTM,
		logger:      log.New(os.Stdout, "TINYSCHEDULER: ", log.LstdFlags),
		timeout:     time.Second * 1,
	}

	taskExecutedAfterShutdown := false
	task := NewTask(func() error {
		taskExecutedAfterShutdown = true
		return nil
	})

	scheduler.AddTask(time.Second, task)
	go scheduler.Start()
	scheduler.Shutdown()

	time.Sleep(100 * time.Millisecond)

	if taskExecutedAfterShutdown {
		t.Errorf("Task executed after scheduler shutdown")
	}
}

func TestTaskExecutionFrequency(t *testing.T) {
	mockTM := NewMockTimeManager()
	scheduler := TaskScheduler{
		tasks:       make(map[time.Duration][]Task),
		quit:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		done:        make(chan struct{}),
		timeManager: mockTM,
		logger:      log.New(os.Stdout, "TINYSCHEDULER: ", log.LstdFlags),
		timeout:     time.Second * 1,
	}

	executionCount := 0
	var mu sync.Mutex
	task := NewTask(func() error {
		mu.Lock()
		executionCount++
		mu.Unlock()
		return nil
	})

	scheduler.AddTask(time.Second, task)
	go scheduler.Start()

	// Simulate 3 ticks
	for i := 0; i < 3; i++ {
		mockTM.C <- time.Now()
		time.Sleep(50 * time.Millisecond)
	}

	if executionCount != 3 {
		t.Errorf("Expected 3 executions, got %d", executionCount)
	}

	scheduler.Shutdown()
}
