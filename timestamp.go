package sadl

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Timestamp struct {
	time.Time
}

const RFC3339Milli = "%d-%02d-%02dT%02d:%02d:%02d.%03dZ"

func (ts Timestamp) String() string {
	if ts.IsZero() {
		return ""
	}
	return fmt.Sprintf(RFC3339Milli, ts.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond()/1000000)
}

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return []byte("\"" + ts.String() + "\""), nil
}

func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err == nil {
		var tsp Timestamp
		tsp, err = ParseTimestamp(string(j))
		if err == nil {
			*ts = tsp
		}
	}
	return err
}

func ParseTimestamp(s string) (Timestamp, error) {
	layout := "2006-01-02T15:04:05.999Z" //derive this from the spec used for output?
	t, e := time.Parse(layout, s)
	if e != nil {
		if strings.HasSuffix(s, "+00:00") || strings.HasSuffix(s, "-00:00") {
			t, e = time.Parse(layout, s[:len(s)-6]+"Z")
		} else if strings.HasSuffix(s, "+0000") || strings.HasSuffix(s, "-0000") {
			t, e = time.Parse(layout, s[:len(s)-5]+"Z")
		}
		if e != nil {
			var ts Timestamp
			return ts, fmt.Errorf("Bad Timestamp: %q", s)
		}
	}
	return Timestamp{t}, nil
}
