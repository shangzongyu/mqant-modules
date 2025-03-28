// Copyright 2014 loolgame Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package room

import (
	"time"

	"github.com/shangzongyu/mqant/gate"
)

type BasePlayerImp struct {
	session      gate.Session
	lastNewsDate int64 //玩家最后一次成功通信时间	单位秒
	body         interface{}
}

func (bp *BasePlayerImp) Type() string {
	return "BasePlayer"
}

func (bp *BasePlayerImp) IsBind() bool {
	if bp.session == nil {
		return false
	} else {
		return true
	}
}

func (bp *BasePlayerImp) Bind(session gate.Session) BasePlayer {
	bp.lastNewsDate = time.Now().Unix()
	bp.session = session
	return bp
}

func (bp *BasePlayerImp) UnBind() error {
	bp.session = nil
	return nil
}

// OnRequest 玩家主动发请求时间
func (bp *BasePlayerImp) OnRequest(session gate.Session) {
	bp.session = session
	bp.lastNewsDate = time.Now().Unix()
}

// OnResponse 服务器主动发送消息给客户端的时间
func (bp *BasePlayerImp) OnResponse(session gate.Session) {
	bp.session = session
	bp.lastNewsDate = time.Now().Unix()
}

func (bp *BasePlayerImp) GetLastReqResDate() int64 {
	return bp.lastNewsDate
}
func (bp *BasePlayerImp) Body() interface{} {
	return bp.body
}

func (bp *BasePlayerImp) SetBody(body interface{}) {
	bp.body = body
}

func (bp *BasePlayerImp) Session() gate.Session {
	return bp.session
}
