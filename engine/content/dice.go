package content

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

// Dice is a roll: Count dice of Sides faces plus Bonus, written "3d10+8", "2d2" or "5".
type Dice struct {
	Count, Sides, Bonus int
}

var diceRE = regexp.MustCompile(`^(?:(\d+)d(\d+))?(?:\+?(\d+))?$`)

// ParseDice reads dice written as "NdM+K", "NdM" or "K".
func ParseDice(s string) (Dice, error) {
	m := diceRE.FindStringSubmatch(s)
	if s == "" || m == nil {
		return Dice{}, fmt.Errorf("bad dice %q", s)
	}
	var d Dice
	d.Count, _ = strconv.Atoi(m[1])
	d.Sides, _ = strconv.Atoi(m[2])
	d.Bonus, _ = strconv.Atoi(m[3])
	if m[1] != "" && d.Sides == 0 {
		return Dice{}, fmt.Errorf("bad dice %q: no sides", s)
	}
	return d, nil
}

func (d Dice) String() string {
	switch {
	case d.Count == 0:
		return strconv.Itoa(d.Bonus)
	case d.Bonus == 0:
		return fmt.Sprintf("%dd%d", d.Count, d.Sides)
	}
	return fmt.Sprintf("%dd%d+%d", d.Count, d.Sides, d.Bonus)
}

// Max is the highest the dice can roll.
func (d Dice) Max() int { return d.Count*d.Sides + d.Bonus }

func (d Dice) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

func (d *Dice) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := ParseDice(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Duration is a time in milliseconds, or a range to pick from each time. In JSON it is a number,
// or [min, max] when Min and Max differ.
type Duration struct {
	Min, Max int
}

// Fixed returns a duration of ms milliseconds.
func Fixed(ms int) Duration { return Duration{ms, ms} }

func (d Duration) MarshalJSON() ([]byte, error) {
	if d.Min == d.Max {
		return json.Marshal(d.Min)
	}
	return json.Marshal([2]int{d.Min, d.Max})
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var ms int
	if err := json.Unmarshal(data, &ms); err == nil {
		*d = Fixed(ms)
		return nil
	}
	var r [2]int
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("duration: want milliseconds or [min, max]")
	}
	*d = Duration{r[0], r[1]}
	return nil
}
