package handler

import "github.com/vetrovegor/kushfinds-backend/internal/location/state"

type StatesResponse struct {
	States []state.State `json:"states"`
}
