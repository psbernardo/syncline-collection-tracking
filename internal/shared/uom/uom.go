package uom

import "strings"

type Code string

const (
	PC   Code = "PC"
	CASE Code = "CASE"
	PACK Code = "PACK"
	BTL  Code = "BTL"
	REAM Code = "REAM"
	ROLL Code = "ROLL"
	BOX  Code = "BOX"
)

func Normalize(value string) Code { return Code(strings.ToUpper(strings.TrimSpace(value))) }

func Valid(value Code) bool {
	switch value {
	case PC, CASE, PACK, BTL, REAM, ROLL, BOX:
		return true
	default:
		return false
	}
}

func Options() []Code { return []Code{PC, CASE, PACK, BTL, REAM, ROLL, BOX} }
