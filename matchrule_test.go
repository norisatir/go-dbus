package dbus

import (
	"testing"
)

func TestToString(t *testing.T) {
	verifyStr := "type='signal',interface='org.freedesktop.DBus',member='Foo',path='/bar/foo'"

	mr := MatchRule{
		Type:      TypeSignal,
		Interface: "org.freedesktop.DBus",
		Member:    "Foo",
		Path:      "/bar/foo"}

	if mr.String() != verifyStr {
		t.Error("#1 Failed")
	}
}
