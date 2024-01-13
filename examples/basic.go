package main

import (
	"github.com/onerciller/tinyscheduler"
	"time"
)

func main() {
	sch := tinyscheduler.NewTaskScheduler()
	task := tinyscheduler.NewTask(func() error {
		println("Hello world!")
		return nil
	})

	sch.AddTask(time.Second*1, task)
	sch.ListenForGracefullyShutdown()
	sch.Start()
}
