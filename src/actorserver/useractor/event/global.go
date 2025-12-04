package event

import "regexp"

// There are some default priority constants
const (
	Min         = -300
	Low         = -200
	BelowNormal = -100
	Normal      = 0
	AboveNormal = 100
	High        = 200
	Max         = 300
)
const Wildcard = "*"

// regex for check good event name.
var eventNameReg = regexp.MustCompile(`^[a-zA-Z][\w-.*]*$`)

// M is short name for map[string]interface{}
type M = map[string]interface{}

// GlobalManager global event manager
var GlobalManager = NewManager("global")
