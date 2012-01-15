package dbus

import "reflect"
import "fmt"
import "strings"

// Matches all messages with equal type, interface, member, or path.
// Any missing/invalid fields are not matched against.
type MatchRule struct {
	Type      MessageType
	Interface string
	Member    string
	Path      string
}

// A string representation af the MatchRule (D-Bus variant map).
func (p *MatchRule) String() string {
	strslice := []string{}

	v := reflect.Indirect(reflect.ValueOf(p))
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		strslice = append(strslice, (fmt.Sprintf("%s='%v'", strings.ToLower(t.Field(i).Name), v.Field(i).Interface())))
	}

	return strings.Join(strslice, ",")
}

func (p *MatchRule) _Match(msg *Message) bool {
	if p.Type != TypeInvalid && p.Type != msg.Type {
		return false
	}
	if p.Interface != "" && p.Interface != msg.Iface {
		return false
	}
	if p.Member != "" && p.Member != msg.Member {
		return false
	}
	if p.Path != "" && p.Path != msg.Path {
		return false
	}
	return true
}
