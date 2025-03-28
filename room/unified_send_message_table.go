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
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/shangzongyu/mqant/gate"
    "github.com/shangzongyu/mqant/log"
    "github.com/shangzongyu/mqant/module"
    "github.com/yireyun/go-queue"
)

type Filter func()

type CallBackMsg struct {
    notify    bool     //是否是广播
    needReply bool     //是否需要回复
    players   []string //如果不是广播就指定session
    topic     *string
    body      *[]byte
}
type TableImp interface {
    GetSeats() map[string]BasePlayer
    GetApp() module.App
}
type UnifiedSendMessageTable struct {
    queue_message *queue.EsQueue
    tableimp      TableImp
}

func (this *UnifiedSendMessageTable) UnifiedSendMessageTableInit(tableimp TableImp, Capaciity uint32) {
    this.queue_message = queue.NewQueue(Capaciity)
    this.tableimp = tableimp
}
func (this *UnifiedSendMessageTable) FindPlayer(session gate.Session) BasePlayer {
    for _, player := range this.tableimp.GetSeats() {
        if (player != nil) && (player.Session() != nil) {
            if player.Session().IsGuest() {
                if player.Session().GetSessionId() == session.GetSessionId() {
                    return player
                }
            } else {
                if player.Session().GetUserId() == session.GetUserId() {
                    return player
                }
            }
        }
    }
    return nil
}

func (this *UnifiedSendMessageTable) SendCallBackMsg(players []string, topic string, body []byte) error {
    ok, quantity := this.queue_message.Put(&CallBackMsg{
        notify:    false,
        needReply: true,
        players:   players,
        topic:     &topic,
        body:      &body,
    })
    if !ok {
        return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
    } else {
        return nil
    }
}

func (this *UnifiedSendMessageTable) NotifyCallBackMsg(topic string, body []byte) error {
    ok, quantity := this.queue_message.Put(&CallBackMsg{
        notify:    true,
        needReply: true,
        players:   nil,
        topic:     &topic,
        body:      &body,
    })
    if !ok {
        return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
    } else {
        return nil
    }
}

func (this *UnifiedSendMessageTable) SendCallBackMsgNR(players []string, topic string, body []byte) error {
    ok, quantity := this.queue_message.Put(&CallBackMsg{
        notify:    false,
        needReply: false,
        players:   players,
        topic:     &topic,
        body:      &body,
    })
    if !ok {
        return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
    } else {
        return nil
    }
}

func (this *UnifiedSendMessageTable) NotifyCallBackMsgNR(topic string, body []byte) error {
    ok, quantity := this.queue_message.Put(&CallBackMsg{
        notify:    true,
        needReply: false,
        players:   nil,
        topic:     &topic,
        body:      &body,
    })
    if !ok {
        return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
    } else {
        return nil
    }
}

func (this *UnifiedSendMessageTable) SendRealMsg(players []string, topic string, body []byte) error {
    this.SendMsg(nil, &CallBackMsg{
        notify:    false,
        needReply: true,
        players:   players,
        topic:     &topic,
        body:      &body,
    })
    return nil
}

func (this *UnifiedSendMessageTable) NotifyRealMsg(topic string, body []byte) error {
    this.SendMsg(nil, &CallBackMsg{
        notify:    true,
        needReply: true,
        players:   nil,
        topic:     &topic,
        body:      &body,
    })
    return nil
}

func (this *UnifiedSendMessageTable) SendRealMsgNR(players []string, topic string, body []byte) error {
    this.SendMsg(nil, &CallBackMsg{
        notify:    false,
        needReply: false,
        players:   players,
        topic:     &topic,
        body:      &body,
    })
    return nil
}

func (this *UnifiedSendMessageTable) NotifyRealMsgNR(topic string, body []byte) error {
    this.SendMsg(nil, &CallBackMsg{
        notify:    true,
        needReply: false,
        players:   nil,
        topic:     &topic,
        body:      &body,
    })
    return nil
}

/*
*
合并玩家所在网关
*/
func (this *UnifiedSendMessageTable) mergeGate() map[string][]string {
    merge := map[string][]string{}
    for _, role := range this.tableimp.GetSeats() {
        if role != nil && role.Session() != nil {
            //未断网
            if _, ok := merge[role.Session().GetServerID()]; ok {
                merge[role.Session().GetServerID()] = append(merge[role.Session().GetServerID()], role.Session().GetSessionId())
            } else {
                merge[role.Session().GetServerID()] = []string{role.Session().GetSessionID()}
            }
        }
    }
    return merge
}

/*
*
【每帧调用】统一发送所有消息给各个客户端
*/
func (this *UnifiedSendMessageTable) ExecuteCallBackMsg(span log.TraceSpan) {
    ok := true
    queue := this.queue_message
    var index = 0
    for ok {
        val, _ok, _ := queue.Get()
        index++
        if _ok {
            msg := val.(*CallBackMsg)
            this.SendMsg(span, msg)
        }
        ok = _ok
    }
}

/*
*
统一发送所有消息给各个客户端
*/
func (this *UnifiedSendMessageTable) SendMsg(span log.TraceSpan, msg *CallBackMsg) {
    if msg.notify {
        merge := this.mergeGate()
        for serverid, plist := range merge {
            sessionids := strings.Join(plist, ",")
            server, e := this.tableimp.GetApp().GetServerByID(serverid)
            if e != nil {
                log.Warning("SendBatch error %v", e)
                return
            }
            if msg.needReply {
                ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
                result, err := server.Call(ctx, "SendBatch", span, sessionids, *msg.topic, *msg.body)
                if err != "" {
                    log.Warning("SendBatch error %v %v", serverid, err)
                } else {
                    if int(result.(int64)) < len(plist) {
                        //有连接断了
                    }
                }
            } else {
                err := server.CallNR("SendBatch", span, sessionids, *msg.topic, *msg.body)
                if err != nil {
                    log.Warning("SendBatch error %v %v", serverid, err.Error())
                }
            }

        }
    } else {
        for _, sessionId := range msg.players {
            for _, role := range this.tableimp.GetSeats() {
                if role != nil {
                    if (role.Session() != nil) && (role.Session().GetSessionID() == sessionId) {
                        if msg.needReply {
                            e := role.Session().Send(*msg.topic, *msg.body)
                            if e == "" {
                                role.OnResponse(role.Session())
                            }
                        } else {
                            _ = role.Session().SendNR(*msg.topic, *msg.body)
                        }
                    }
                }

            }
        }
    }
}
