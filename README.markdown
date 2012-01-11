Documentation
=============

Look at the API on [GoPkgDoc](http://gopkgdoc.appspot.com/pkg/github.com/norisatir/go-dbus).

Installation
============

    goinstall github.com/norisatir/go-dbus

Usage
=====

Methods
-------

Methods is obtained with

    meth, err := conn.Object(dest, path).Interface(iface).Method(method)

They are called with

    out, err := conn.Call(meth)

Signals
-------

Signals are obtained with

    sig, err := conn.Object(dest, path).Interface(iface).Signal(signal)

they are emitted with

    err = conn.Emit(sig)

**TODO** Add signal handling usage.

An example
----------

```go
// Issue OSD notifications according to the Desktop Notifications Specification 1.1
//      http://people.canonical.com/~agateau/notifications-1.1/spec/index.html
// See also
//      https://wiki.ubuntu.com/NotifyOSD#org.freedesktop.Notifications.Notify
package main

import "github.com/norisatir/go-dbus"
import "log"

func main() {
    var (
        err error
        conn *dbus.Connection
        method *dbus.Method
        out []interface{}
    )

    // Connect to Session or System buses.
    if conn, err = dbus.Connect(dbus.SessionBus); err != nil {
        log.Fatal("Connection error:", err)
    }
    if err = conn.Initialize(); err != nil {
        log.Fatal("Initialization error:", err)
    }

    // Get objects.
    obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

    // Introspect objects.
    var intro dbus.Introspect
    method, err = obj.Interface("org.freedesktop.DBus.Introspectable").Method("Introspect")
    if err != nil {
        log.Fatal(err)
    }
    out, err = conn.Call(method)
    if err != nil {
        log.Fatal("Introspect error:", err)
    }
    intro, err = dbus.NewIntrospect(out[0].(string))
    m := intro.GetInterfaceData("org.freedesktop.Notifications").GetMethodData("Notify")
    log.Printf("%s in:%s out:%s", m.GetName(), m.GetInSignature(), m.GetOutSignature())

    // Call object methods.
    method, err = obj.Interface("org.freedesktop.Notifications").Method("Notify")
    if err != nil {
        log.Fatal(err)
    }
    out, err = conn.Call(method,
		"dbus-tutorial", uint32(0), "",
        "dbus-tutorial", "You've been notified!",
		[]interface{}{}, map[string]interface{}{}, int32(-1))
    if err != nil {
        log.Fatal("Notification error:", err)
    }
    log.Print("Notification id:", out[0])
}
```
