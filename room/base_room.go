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
	"sync"

	"github.com/shangzongyu/mqant/module"
)

type Room struct {
	app    module.App
	tables sync.Map
	roomId int
}

type NewTableFunc func(module module.App, tableId string) (BaseTable, error)

func NewRoom(module module.App) *Room {
	room := &Room{
		app: module,
	}
	return room
}

func (ro *Room) OnInit(module module.App, roomId int) error {
	ro.app = module
	ro.roomId = roomId
	return nil
}

func (ro *Room) RoomId() int {
	return ro.roomId
}

func (ro *Room) CreateById(app module.App, tableId string, newTablefunc NewTableFunc) (BaseTable, error) {
	if table, ok := ro.tables.Load(tableId); ok {
		table.(BaseTable).Run()
		return table.(BaseTable), nil
	}
	table, err := newTablefunc(app, tableId)
	if err != nil {
		return nil, err
	}
	ro.tables.Store(table.TableId(), table)
	return table, nil
}

func (ro *Room) GetTable(tableId string) BaseTable {
	if table, ok := ro.tables.Load(tableId); ok {
		table.(BaseTable).Run()
		return table.(BaseTable)
	}
	return nil
}

func (ro *Room) DestroyTable(tableId string) error {
	ro.tables.Delete(tableId)
	return nil
}
