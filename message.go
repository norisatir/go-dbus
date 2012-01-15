package dbus

import (
	"bytes"
	"sync"
)

// See the D-Bus tutorial for information about message types.
//		http://dbus.freedesktop.org/doc/dbus-tutorial.html#messages
type MessageType uint8

const (
	TypeInvalid MessageType = iota
	TypeMethodCall
	TypeMethodReturn
	TypeError
	TypeSignal
)

var messageTypeString = map[MessageType]string{
	TypeInvalid:      "invalid",
	TypeMethodCall:   "method_call",
	TypeMethodReturn: "method_return",
	TypeSignal:       "signal",
	TypeError:        "error",
}

func (t MessageType) String() string { return messageTypeString[t] }

type MessageFlag uint8

const (
	FlagNoReplyExpected MessageFlag = 1 << iota
	FlagNoAutoStart
)

type Message struct {
	Type        MessageType
	Flags       MessageFlag
	Protocol    int
	bodyLength  int
	Path        string
	Dest        string
	Iface       string
	Member      string
	Sig         string
	Params      []interface{}
	serial      int
	replySerial uint32
	ErrorName   string
	//	Sender;
}

var serialMutex sync.Mutex
var messageSerial = int(0)

func _GetNewSerial() int {
	serialMutex.Lock()
	messageSerial++
	serial := messageSerial
	serialMutex.Unlock()
	return serial
}

// Create a new message with Flags == 0 and Protocol == 1.
func NewMessage() *Message {
	msg := new(Message)

	msg.serial = _GetNewSerial()
	msg.replySerial = 0
	msg.Flags = 0
	msg.Protocol = 1

	msg.Params = make([]interface{}, 0)

	return msg
}

func (p *Message) _BufferToMessage(buff []byte) (int, error) {
	slice, bufIdx, e := Parse(buff, "yyyyuua(yv)", 0)
	if e != nil {
		return 0, e
	}

	p.Type = MessageType(slice[1].(byte))
	p.Flags = MessageFlag(slice[2].(byte))
	p.Protocol = int(slice[3].(byte))
	p.bodyLength = int(slice[4].(uint32))
	p.serial = int(slice[5].(uint32))

	if vec, ok := slice[6].([]interface{}); ok {
		for _, v := range vec {
			tmpSlice := v.([]interface{})
			t := int(tmpSlice[0].(byte))
			val := tmpSlice[1]

			switch t {
			case 1:
				p.Path = val.(string)
			case 2:
				p.Iface = val.(string)
			case 3:
				p.Member = val.(string)
			case 4:
				p.ErrorName = val.(string)
			case 5:
				p.replySerial = val.(uint32)
			case 6:
				p.Dest = val.(string)
			case 7:
				// FIXME
			case 8:
				p.Sig = val.(string)
			}
		}
	}
	idx := _Align(8, bufIdx)
	if 0 < p.bodyLength {
		p.Params, idx, _ = Parse(buff, p.Sig, idx)
	}
	return idx, nil
}

func _Unmarshal(buff []byte) (*Message, int, error) {
	msg := NewMessage()
	idx, e := msg._BufferToMessage(buff)
	if e != nil {
		return nil, 0, e
	}
	return msg, idx, nil
}

func (p *Message) _Marshal() ([]byte, error) {
	buff := bytes.NewBuffer([]byte{})
	_AppendByte(buff, byte('l')) // little Endian
	_AppendByte(buff, byte(p.Type))
	_AppendByte(buff, byte(p.Flags))
	_AppendByte(buff, byte(p.Protocol))

	tmpBuff := bytes.NewBuffer([]byte{})
	_AppendParamsData(tmpBuff, p.Sig, p.Params)
	_AppendUint32(buff, uint32(len(tmpBuff.Bytes())))
	_AppendUint32(buff, uint32(p.serial))

	_AppendArray(buff, 1,
		func(b *bytes.Buffer) {
			if p.Path != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 1) // path
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 'o')
				_AppendByte(b, 0)
				_AppendString(b, p.Path)
			}

			if p.Iface != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 2) // interface
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 's')
				_AppendByte(b, 0)
				_AppendString(b, p.Iface)
			}

			if p.Member != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 3) // member
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 's')
				_AppendByte(b, 0)
				_AppendString(b, p.Member)
			}

			if p.replySerial != 0 {
				_AppendAlign(8, b)
				_AppendByte(b, 5) // reply serial
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 'u')
				_AppendByte(b, 0)
				_AppendUint32(b, uint32(p.replySerial))
			}

			if p.Dest != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 6) // destination
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 's')
				_AppendByte(b, 0)
				_AppendString(b, p.Dest)
			}

			if p.Sig != "" {
				_AppendAlign(8, b)
				_AppendByte(b, 8) // signature
				_AppendByte(b, 1) // signature size
				_AppendByte(b, 'g')
				_AppendByte(b, 0)
				_AppendSignature(b, p.Sig)
			}
		})

	_AppendAlign(8, buff)
	_AppendParamsData(buff, p.Sig, p.Params)

	return buff.Bytes(), nil
}
