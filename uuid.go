package sadl

import (
	"encoding/json"
	"fmt"
)

type UUID string

func (u *UUID) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err == nil {
		v := Parse(string(j))
		if v != "" {
			*u = v
			return nil
		}
	}
	return fmt.Errorf("Bad UUID: %v", string(b))
}

const Length = 36

func Parse(text string) UUID {
	if len(text) != Length {
		return UUID("")
	}
	if text[8] != '-' || text[13] != '-' || text[18] != '-' || text[23] != '-' {
		return ""
	}
	return UUID(text)
}
