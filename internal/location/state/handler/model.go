package handler

import "github.com/xw1nchester/kushfinds-backend/internal/location/state"

type StatesResponse struct {
	States []state.State `json:"states"`
}
