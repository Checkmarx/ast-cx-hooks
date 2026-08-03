package cursor

import (
	"encoding/json"
	"math"
)

// Milliseconds is a duration in milliseconds. Cursor documents duration fields as
// "number" and may send integer or floating-point values (e.g. 2850.456).
type Milliseconds int64

// Int64 returns the duration truncated to the nearest millisecond.
func (m Milliseconds) Int64() int64 { return int64(m) }

// UnmarshalJSON accepts JSON numbers whether encoded as integers or floats.
func (m *Milliseconds) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	f, err := n.Float64()
	if err != nil {
		return err
	}
	*m = Milliseconds(int64(math.Round(f)))
	return nil
}
