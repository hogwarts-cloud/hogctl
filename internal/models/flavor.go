package models

type Flavor string

var AvailableFlavors []Flavor = []Flavor{"micro"}

func (f Flavor) CPU() string {
	switch f {
	case "micro":
		return "1"
	default:
		return "undefined"
	}
}

func (f Flavor) Memory() string {
	switch f {
	case "micro":
		return "1"
	default:
		return "undefined"
	}
}
