package dbus

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

type StandardBus int

const (
	SessionBus StandardBus = iota
	SystemBus
)

const dbusXMLIntro = `
<!DOCTYPE node PUBLIC "-//freedesktop//DTD D-BUS Object Introspection 1.0//EN"
"http://www.freedesktop.org/standards/dbus/1.0/introspect.dtd">
<node>
  <interface name="org.freedesktop.DBus.Introspectable">
    <method name="Introspect">
      <arg name="data" direction="out" type="s"/>
    </method>
  </interface>
  <interface name="org.freedesktop.DBus">
    <method name="RequestName">
      <arg direction="in" type="s"/>
      <arg direction="in" type="u"/>
      <arg direction="out" type="u"/>
    </method>
    <method name="ReleaseName">
      <arg direction="in" type="s"/>
      <arg direction="out" type="u"/>
    </method>
    <method name="StartServiceByName">
      <arg direction="in" type="s"/>
      <arg direction="in" type="u"/>
      <arg direction="out" type="u"/>
    </method>
    <method name="Hello">
      <arg direction="out" type="s"/>
    </method>
    <method name="NameHasOwner">
      <arg direction="in" type="s"/>
      <arg direction="out" type="b"/>
    </method>
    <method name="ListNames">
      <arg direction="out" type="as"/>
    </method>
    <method name="ListActivatableNames">
      <arg direction="out" type="as"/>
    </method>
    <method name="AddMatch">
      <arg direction="in" type="s"/>
    </method>
    <method name="RemoveMatch">
      <arg direction="in" type="s"/>
    </method>
    <method name="GetNameOwner">
      <arg direction="in" type="s"/>
      <arg direction="out" type="s"/>
    </method>
    <method name="ListQueuedOwners">
      <arg direction="in" type="s"/>
      <arg direction="out" type="as"/>
    </method>
    <method name="GetConnectionUnixUser">
      <arg direction="in" type="s"/>
      <arg direction="out" type="u"/>
    </method>
    <method name="GetConnectionUnixProcessID">
      <arg direction="in" type="s"/>
      <arg direction="out" type="u"/>
    </method>
    <method name="GetConnectionSELinuxSecurityContext">
      <arg direction="in" type="s"/>
      <arg direction="out" type="ay"/>
    </method>
    <method name="ReloadConfig">
    </method>
    <signal name="NameOwnerChanged">
      <arg type="s"/>
      <arg type="s"/>
      <arg type="s"/>
    </signal>
    <signal name="NameLost">
      <arg type="s"/>
    </signal>
    <signal name="NameAcquired">
      <arg type="s"/>
    </signal>
  </interface>
</node>`

type signalHandler struct {
	mr   MatchRule
	proc func(*Message)
}

type Connection struct {
	addressMap        map[string]string
	uniqName          string
	methodCallReplies map[uint32](func(msg *Message))
	signalMatchRules  []signalHandler
	conn              net.Conn
	buffer            *bytes.Buffer
	proxy             *Interface
}

type Object struct {
	dest  string
	path  string
	intro Introspect
}

type Interface struct {
	obj   *Object
	name  string
	intro InterfaceData
}

type Method struct {
	iface *Interface
	data  MethodData
}

type Signal struct {
	iface *Interface
	data  SignalData
}

// Retrieve a method by name.
func (iface *Interface) Method(name string) (*Method, error) {
	method := iface.intro.GetMethodData(name)
	if nil == method {
		return nil, errors.New("Invalid Method")
	}
	return &Method{iface, method}, nil
}

// Retrieve a signal by name.
func (iface *Interface) Signal(name string) (*Signal, error) {
	signal := iface.intro.GetSignalData(name)
	if nil == signal {
		return nil, errors.New("Invalid Signalx")
	}
	return &Signal{iface, signal}, nil
}

func Connect(busType StandardBus) (*Connection, error) {
	var address string

	switch busType {
	case SessionBus:
		address = os.Getenv("DBUS_SESSION_BUS_ADDRESS")

	case SystemBus:
		if address = os.Getenv("DBUS_SYSTEM_BUS_ADDRESS"); len(address) == 0 {
			address = "unix:path=/var/run/dbus/system_bus_socket"
		}

	default:
		return nil, errors.New("Unknown bus")
	}

	if len(address) == 0 {
		return nil, errors.New("Unknown bus address")
	}
	transport := address[:strings.Index(address, ":")]

	bus := new(Connection)
	bus.addressMap = make(map[string]string)
	for _, pair := range strings.Split(address[len(transport)+1:], ",") {
		pair := strings.Split(pair, "=")
		bus.addressMap[pair[0]] = pair[1]
	}

	var ok bool
	if address, ok = bus.addressMap["path"]; ok {
	} else if address, ok = bus.addressMap["abstract"]; ok {
		address = "@" + address
	} else {
		return nil, errors.New("Unknown address key")
	}

	var err error
	if bus.conn, err = net.Dial(transport, address); err != nil {
		return nil, err
	}

	if _, err = bus.conn.Write([]byte{0}); err != nil {
		return nil, err
	}

	bus.methodCallReplies = make(map[uint32]func(*Message))
	bus.signalMatchRules = make([]signalHandler, 0)
	bus.proxy = bus._GetProxy()
	bus.buffer = bytes.NewBuffer([]byte{})
	return bus, nil
}

func (p *Connection) Authenticate() error {
	if err := p._Authenticate(new(AuthDbusCookieSha1)); err != nil {
		return err
	}
	go p._RunLoop()
	p._SendHello()
	return nil
}

func (p *Connection) _MessageReceiver(msgChan chan *Message) {
	for {
		msg, e := p._PopMessage()
		if e == nil {
			msgChan <- msg
			continue // might be another msg in p.buffer
		}
		p._UpdateBuffer()
	}
}

func (p *Connection) _RunLoop() {
	msgChan := make(chan *Message)
	go p._MessageReceiver(msgChan)
	for {
		select {
		case msg := <-msgChan:
			p._MessageDispatch(msg)
		}
	}
}

func (p *Connection) _MessageDispatch(msg *Message) {
	if msg == nil {
		return
	}

	switch msg.Type {
	case METHOD_RETURN:
		rs := msg.replySerial
		if replyFunc, ok := p.methodCallReplies[rs]; ok {
			replyFunc(msg)
			delete(p.methodCallReplies, rs)
		}
	case SIGNAL:
		for _, handler := range p.signalMatchRules {
			if handler.mr._Match(msg) {
				handler.proc(msg)
			}
		}
	case ERROR:
		fmt.Println("ERROR")
	}
}

func (p *Connection) _PopMessage() (*Message, error) {
	msg, n, err := _Unmarshal(p.buffer.Bytes())
	if err != nil {
		return nil, err
	}
	p.buffer.Read(make([]byte, n)) // remove first n bytes
	return msg, nil
}

func (p *Connection) _UpdateBuffer() error {
	//	_, e := p.buffer.ReadFrom(p.conn);
	buff := make([]byte, 4096)
	n, e := p.conn.Read(buff)
	p.buffer.Write(buff[0:n])
	return e
}

func (p *Connection) _SendSync(msg *Message, callback func(*Message)) error {
	seri := uint32(msg.serial)
	recvChan := make(chan int)
	p.methodCallReplies[seri] = func(rmsg *Message) {
		callback(rmsg)
		recvChan <- 0
	}

	buff, _ := msg._Marshal()
	p.conn.Write(buff)
	<-recvChan // synchronize
	return nil
}

func (p *Connection) _SendHello() error {
	if method, err := p.proxy.Method("Hello"); err == nil {
		p.Call(method)
	}
	return nil
}

func (p *Connection) _GetIntrospect(dest string, path string) Introspect {
	msg := NewMessage()
	msg.Type = METHOD_CALL
	msg.Path = path
	msg.Dest = dest
	msg.Iface = "org.freedesktop.DBus.Introspectable"
	msg.Member = "Introspect"

	var intro Introspect

	p._SendSync(msg, func(reply *Message) {
		if v, ok := reply.Params[0].(string); ok {
			if i, err := NewIntrospect(v); err == nil {
				intro = i
			}
		}
	})

	return intro
}

// Retrieve an interface by name.
func (obj *Object) Interface(name string) *Interface {
	if obj == nil || obj.intro == nil {
		return nil
	}

	iface := new(Interface)
	iface.obj = obj
	iface.name = name

	data := obj.intro.GetInterfaceData(name)
	if nil == data {
		return nil
	}

	iface.intro = data

	return iface
}

func (p *Connection) _GetProxy() *Interface {
	obj := new(Object)
	obj.path = "/org/freedesktop/DBus"
	obj.dest = "org.freedesktop.DBus"
	obj.intro, _ = NewIntrospect(dbusXMLIntro)

	iface := new(Interface)
	iface.obj = obj
	iface.name = "org.freedesktop.DBus"
	iface.intro = obj.intro.GetInterfaceData("org.freedesktop.DBus")

	return iface
}

// Call a method with the given arguments.
func (p *Connection) Call(method *Method, args ...interface{}) ([]interface{}, error) {
	iface := method.iface
	msg := NewMessage()

	msg.Type = METHOD_CALL
	msg.Path = iface.obj.path
	msg.Iface = iface.name
	msg.Dest = iface.obj.dest
	msg.Member = method.data.GetName()
	msg.Sig = method.data.GetInSignature()
	if len(args) > 0 {
		msg.Params = args[:]
	}

	var ret []interface{}
	p._SendSync(msg, func(reply *Message) {
		ret = reply.Params
	})

	return ret, nil
}

// Emit a signal with the given arguments.
func (p *Connection) Emit(signal *Signal, args ...interface{}) error {
	iface := signal.iface

	msg := NewMessage()

	msg.Type = SIGNAL
	msg.Path = iface.obj.path
	msg.Iface = iface.name
	msg.Dest = iface.obj.dest
	msg.Member = signal.data.GetName()
	msg.Sig = signal.data.GetSignature()
	msg.Params = args[:]

	buff, _ := msg._Marshal()
	_, err := p.conn.Write(buff)

	return err
}

// Retrieve a specified object.
func (p *Connection) Object(dest string, path string) *Object {

	obj := new(Object)
	obj.path = path
	obj.dest = dest
	obj.intro = p._GetIntrospect(dest, path)

	return obj
}

// Handle received signals.
func (p *Connection) Handle(rule *MatchRule, handler func(*Message)) {
	p.signalMatchRules = append(p.signalMatchRules, signalHandler{*rule, handler})
	if method, err := p.proxy.Method("AddMatch"); err == nil {
		p.Call(method, rule._ToString())
	}
}
