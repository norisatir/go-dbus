package dbus

import "testing"

func TestUnmarshal(t *testing.T) {

	teststr := "l\x01\x00\x01\x00\x00\x00\x00\x01\x00\x00\x00m\x00\x00\x00\x01\x01o\x00\x15\x00\x00\x00/org/freedesktop/DBus\x00\x00\x00\x02\x01s\x00\x14\x00\x00\x00org.freedesktop.DBus\x00\x00\x00\x00\x03\x01s\x00\x05\x00\x00\x00Hello\x00\x00\x00\x06\x01s\x00\x14\x00\x00\x00org.freedesktop.DBus\x00\x00\x00\x00"

	msg, _, e := _Unmarshal([]byte(teststr))
	if nil != e {
		t.Error("Unmarshal Failed")
	}
	if TypeMethodCall != msg.Type {
		t.Error("#1 Failed :", msg.Type)
	}
	if "/org/freedesktop/DBus" != msg.Path {
		t.Error("#2 Failed :", msg.Path)
	}
	if "org.freedesktop.DBus" != msg.Dest {
		t.Error("#3 Failed :", msg.Dest)
	}
	if "org.freedesktop.DBus" != msg.Iface {
		t.Error("#4 Failed :", msg.Iface)
	}
	if "Hello" != msg.Member {
		t.Error("#5 Failed :", msg.Member)
	}
}

func TestMarshal(t *testing.T) {
	teststr := "l\x01\x00\x01\x00\x00\x00\x00\x01\x00\x00\x00m\x00\x00\x00\x01\x01o\x00\x15\x00\x00\x00/org/freedesktop/DBus\x00\x00\x00\x02\x01s\x00\x14\x00\x00\x00org.freedesktop.DBus\x00\x00\x00\x00\x03\x01s\x00\x05\x00\x00\x00Hello\x00\x00\x00\x06\x01s\x00\x14\x00\x00\x00org.freedesktop.DBus\x00\x00\x00\x00"

	msg := NewMessage()
	msg.Type = TypeMethodCall
	msg.Flags = MessageFlag(0)
	msg.Path = "/org/freedesktop/DBus"
	msg.Dest = "org.freedesktop.DBus"
	msg.Iface = "org.freedesktop.DBus"
	msg.Member = "Hello"
	msg.serial = 1

	buff, _ := msg._Marshal()
	if teststr != string(buff) {
		t.Error("#1 Failed\n", buff, "\n", []byte(teststr))
	}
}
