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

import "time"

// TimeOutTable table超时处理机制
type TimeOutTable struct {
	subtable              SubTable
	timeout               int64 //默认超时时间单位秒
	lastCommunicationDate int64
}

func (tot *TimeOutTable) TimeOutTableInit(subtable SubTable, timeout int64) {
	tot.subtable = subtable
	tot.timeout = timeout
	tot.lastCommunicationDate = time.Now().Unix()
}

func (tot *TimeOutTable) ResetTimeOut() {
	tot.lastCommunicationDate = time.Now().Unix()
}

/*
*
检查整个table是否已超时

检查规则:
1. 所有玩家超过指定时间未连接
2. 所有玩家网络中断时间超过指定时间(依赖table内会定期广播消息给玩家)
*/
func (tot *TimeOutTable) CheckTimeOut() {
	for _, player := range tot.subtable.GetSeats() {
		if player != nil {
			if tot.lastCommunicationDate < player.GetLastReqResDate() {
				tot.lastCommunicationDate = player.GetLastReqResDate()
			}
		}
	}
	if tot.timeout > 0 {
		if time.Now().Unix() > (tot.lastCommunicationDate + tot.timeout) {
			tot.subtable.OnTimeOut()
		}
	}
}
