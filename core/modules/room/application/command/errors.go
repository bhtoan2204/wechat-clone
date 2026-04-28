package command

import (
	"errors"
	"net/http"

	"wechat-clone/core/shared/pkg/apperr"
)

var (
	ErrRoomFull            = errors.New("room is full")
	ErrRoomNotFound        = errors.New("room not found")
	ErrRoomAlreadyJoined   = errors.New("room already joined")
	ErrRoomAccountNotFound = errors.New("account not found")

	ErrRoomCommandInvalidState = apperr.New("room.invalid_state", "room command is not valid for the current state", http.StatusConflict)
	ErrRoomCommandForbidden    = apperr.New("room.forbidden", "account is not allowed to mutate this room", http.StatusForbidden)
	ErrRoomCommandNotFound     = apperr.New("room.not_found", "room or message was not found", http.StatusNotFound)
)
