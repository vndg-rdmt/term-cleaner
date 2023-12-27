// Cleaner package
// Author (c) 2023 Belousov Daniil

package cleaner

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type TodoCallback = func(ctx context.Context) error

type Cleaner struct {
	mx     sync.Mutex
	todo   []TodoCallback
	amount uint
}

func whisper(msg string) {
	fmt.Println("\033[32m[closer]\033[0m: " + msg)
}

func New(tasks uint) *Cleaner {
	return &Cleaner{
		mx:     sync.Mutex{},
		todo:   make([]TodoCallback, 0, tasks),
		amount: tasks,
	}
}

func (self *Cleaner) Add(f TodoCallback) {
	self.mx.Lock()
	self.todo = append(self.todo, f)
	self.mx.Unlock()
}

func functionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (self *Cleaner) CloseAll(timeout time.Duration) {
	self.mx.Lock()

	whisper("closing/clearing resources")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	defer func() {
		self.todo = make([]TodoCallback, 0, self.amount)
		self.mx.Unlock()
	}()

	var (
		errmsg = make([]error, 0, len(self.todo))
		done   = make(chan bool, 1)
	)

	go func() {
		for _, f := range self.todo {
			whisper("clearing " + functionName(f))
			if err := f(ctx); err != nil {
				errmsg = append(errmsg, err)
			}
		}

		done <- true
	}()

	select {
	case <-done:
		break
	case <-ctx.Done():
		whisper(fmt.Sprintf("shutdown cancelled by context: %v", ctx.Err()))
	}

	if len(errmsg) > 0 {
		whisper(fmt.Sprintf(
			"shutdown done with error(s):\n%s",
			strings.Join(func() []string {

				buffer := make([]string, len(errmsg))
				for i, e := range errmsg {
					buffer[i] = e.Error() + "\n"
				}
				return buffer

			}(), ", "),
		))
	}
}
