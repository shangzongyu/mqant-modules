// Copyright 2014 loolgame Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package room

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"

	"github.com/pkg/errors"
	"github.com/yireyun/go-queue"
)

type QueueMsg struct {
	Func   string
	Params []interface{}
}

type QueueReceive interface {
	Receive(msg *QueueMsg, index int)
}

type QueueTable struct {
	opts          Options
	functions     map[string]reflect.Value
	receive       QueueReceive
	queue0        *queue.EsQueue
	queue1        *queue.EsQueue
	currentWQueue int //当前写的队列
	lock          *sync.RWMutex
}

func (qt *QueueTable) QueueInit(opts ...Option) {
	qt.opts = newOptions(opts...)
	qt.functions = map[string]reflect.Value{}
	qt.queue0 = queue.NewQueue(qt.opts.Capaciity)
	qt.queue1 = queue.NewQueue(qt.opts.Capaciity)
	qt.currentWQueue = 0
	qt.lock = new(sync.RWMutex)
}

func (qt *QueueTable) SetReceive(receive QueueReceive) {
	qt.receive = receive
}

func (qt *QueueTable) Register(id string, f interface{}) {

	if _, ok := qt.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	qt.functions[id] = reflect.ValueOf(f)
}

// PutQueue goroutine 安全,任 goroutine 可调用
func (qt *QueueTable) PutQueue(_func string, params ...interface{}) error {
	q := qt.wqueue()
	qt.lock.Lock()
	ok, quantity := q.Put(&QueueMsg{
		Func:   _func,
		Params: params,
	})
	qt.lock.Unlock()
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}

}

/*
*
切换并且返回读的队列
*/
func (qt *QueueTable) switchqueue() *queue.EsQueue {
	qt.lock.Lock()
	if qt.currentWQueue == 0 {
		qt.currentWQueue = 1
		qt.lock.Unlock()
		return qt.queue0
	} else {
		qt.currentWQueue = 0
		qt.lock.Unlock()
		return qt.queue1
	}

}
func (qt *QueueTable) wqueue() *queue.EsQueue {
	qt.lock.Lock()
	if qt.currentWQueue == 0 {
		qt.lock.Unlock()
		return qt.queue0
	} else {
		qt.lock.Unlock()
		return qt.queue1
	}

}

// ExecuteEvent 【每帧调用】执行队列中的所有事件
func (qt *QueueTable) ExecuteEvent(arge interface{}) {
	ok := true
	queue := qt.switchqueue()
	index := 0
	for ok {
		val, _ok, _ := queue.Get()
		index++
		if _ok {
			if qt.receive != nil {
				qt.receive.Receive(val.(*QueueMsg), index)
			} else {
				msg := val.(*QueueMsg)
				function, ok := qt.functions[msg.Func]
				if !ok {
					//fmt.Println(fmt.Sprintf("Remote function(%s) not found", msg.Func))
					if qt.opts.NoFound != nil {
						fc, err := qt.opts.NoFound(msg)
						if err != nil {
							qt.opts.RecoverHandle(msg, err)
							continue
						}
						function = fc
					} else {
						if qt.opts.RecoverHandle != nil {
							qt.opts.RecoverHandle(msg, errors.Errorf("Remote function(%s) not found", msg.Func))
						}
						continue
					}
				}
				f := function
				in := make([]reflect.Value, len(msg.Params))
				for k, _ := range in {
					switch v2 := msg.Params[k].(type) { //多选语句switch
					case nil:
						in[k] = reflect.Zero(f.Type().In(k))
					default:
						in[k] = reflect.ValueOf(v2)
					}
					//in[k] = reflect.ValueOf(msg.Params[k])
				}
				_runFunc := func() {
					defer func() {
						if r := recover(); r != nil {
							buff := make([]byte, 1024)
							runtime.Stack(buff, false)
							if qt.opts.RecoverHandle != nil {
								qt.opts.RecoverHandle(msg, errors.New(string(buff)))
							}
						}
					}()
					out := f.Call(in)
					if qt.opts.ErrorHandle != nil {
						if len(out) == 1 {
							value, ok := out[0].Interface().(error)
							if ok {
								if value != nil {
									qt.opts.ErrorHandle(msg, value)
								}
							}
						}
					}
				}
				_runFunc()
			}
		}
		ok = _ok
	}
}
