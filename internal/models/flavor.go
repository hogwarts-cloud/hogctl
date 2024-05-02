package models

type Flavor string

var AvailableFlavors []string = []string{"micro"}

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
