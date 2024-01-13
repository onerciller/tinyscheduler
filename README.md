# TinyScheduler
TinyScheduler is a lightweight, flexible task scheduler written in Go. It's designed to manage and execute tasks at specified intervals, providing an easy-to-use API for scheduling and handling task execution.

## Features
- Task Scheduling: Schedule tasks to run at specified intervals.
- Graceful Shutdown: Handles SIGTERM, SIGQUIT, and SIGINT for a graceful shutdown in containerized environments.
- Error Handling: Customizable error handling for each task.
- Concurrency Control: Executes tasks concurrently and manages their lifecycles.
- Logging Support: Built-in logging for task execution and errors.
- Timeout Management: Ability to set timeouts for task execution.

## Installation
```bash
go get github.com/onerciller/tinyscheduler
```

## Usage
```go

scheduler := tinyscheduler.NewTaskScheduler()

task := NewTask(func() error {
    // Task logic here
    return nil
})

scheduler.AddTask(time.Second*30, task)

scheduler.Start()
```


