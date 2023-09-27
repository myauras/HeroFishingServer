package game

import (
	"matchgame/packet"
)

type PlayerActionType int

const (
	NoneAction PlayerActionType = 0
)

func GetActionTypeName(actionType PlayerActionType) string {
	actionTypeNameMap := map[PlayerActionType]string{
		NoneAction: "NoneAction",
	}
	return actionTypeNameMap[actionType]
}

type PlayerAction struct {
	packet.CMDContent
	ID          int
	ActionType  PlayerActionType
	PlayerIndex int
}

func (aciotn *PlayerAction) Parse(common packet.CMDContent) bool {
	m := common.(map[string]interface{})

	if value, ok := m["ID"].(float64); ok {
		aciotn.ID = int(value)
	} else {
		return false
	}
	if value, ok := m["ActionType"].(float64); ok {
		aciotn.ActionType = PlayerActionType(value)
	} else {
		return false
	}
	if value, ok := m["PlayerIndex"].(float64); ok {
		aciotn.PlayerIndex = int(value)
	} else {
		return false
	}
	return true
}
