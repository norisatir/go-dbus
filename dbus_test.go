package dbus

import (
	"fmt"
	"testing"
)

type callTest struct {
	dest, path, iface, method string
	args                      []interface{}
	validate                  func([]interface{}) error
}

var callTests = []callTest{
	{"org.freedesktop.Notifications", "/org/freedesktop/Notifications",
		"org.freedesktop.Notifications", "Notify",
		[]interface{}{
			"go-dbus", uint32(0),
			"info", "testing go-dbus", "test_body",
			[]string{}, map[uint32]interface{}{},
			int32(2000)},
		func([]interface{}) error {
			return nil
		}},
}

func (test callTest) Call(c *Connection) error {
	method, err := c.Object(test.dest, test.path).Interface(test.iface).Method(test.method)
	if err != nil {
		return err
	}
	out, err := c.Call(method, test.args...)
	if err != nil {
		return fmt.Errorf("failed Method.Call: %v", err)
	}
	if err = test.validate(out); err != nil {
		err = fmt.Errorf("failed validation: %v", err)
	}
	return err
}

func TestDBus(t *testing.T) {
	con, err := Connect(SessionBus)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err = con.Initialize(); err != nil {
		t.Fatal("Failed Connection.Initialize:", err.Error())
	}

	for i, test := range callTests {
		err := test.Call(con)
		if err != nil {
			t.Errorf("callTest %d: %v", i, err)
		}
	}
}
