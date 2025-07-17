package types

import (
	"encoding/json"
	"errors"
	"strconv"
)

type IntOrString int

func (i *IntOrString) UnmarshalJSON(b []byte) error {
	var asInt int
	if err := json.Unmarshal(b, &asInt); err == nil {
		*i = IntOrString(asInt)
		return nil
	}

	var asStr string
	if err := json.Unmarshal(b, &asStr); err == nil {
		parsed, err := strconv.Atoi(asStr)
		if err != nil {
			return err
		}
		*i = IntOrString(parsed)
		return nil
	}

	return errors.New("invalid int or string")
}
