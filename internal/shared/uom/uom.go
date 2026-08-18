package uom

import "strings"

type Code string

const (
	PC   Code = "PC"
	PACK Code = "PACK"
	BTL  Code = "BTL"
	REAM Code = "REAM"
	ROLL Code = "ROLL"
	BOX  Code = "BOX"
)

func Normalize(value string) Code { return Code(strings.ToUpper(strings.TrimSpace(value))) }

func Valid(value Code) bool {
	switch value {
	case PC, PACK, BTL, REAM, ROLL, BOX:
		return true
	default:
		return false
	}
}

func Options() []Code { return []Code{PC, PACK, BTL, REAM, ROLL, BOX} }
