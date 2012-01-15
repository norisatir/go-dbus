package dbus

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"
)

type Authenticator interface {
	Mechanism() []byte
	InitialResponse() []byte
	ProcessData([]byte) ([]byte, error)
}

type AuthExternal struct {
}

func (p *AuthExternal) Mechanism() []byte {
	return []byte("EXTERNAL")
}

func (p *AuthExternal) InitialResponse() []byte {
	uid := []byte(strconv.Itoa(os.Getuid()))
	uidHex := make([]byte, hex.EncodedLen(len(uid)))
	hex.Encode(uidHex, uid)
	return uidHex
}

func (p *AuthExternal) ProcessData([]byte) ([]byte, error) {
	return nil, errors.New("Unexpected Response")
}

type AuthDbusCookieSha1 struct {
}

func (p *AuthDbusCookieSha1) Mechanism() []byte {
	return []byte("DBUS_COOKIE_SHA1")
}

func (p *AuthDbusCookieSha1) InitialResponse() []byte {
	user := []byte(os.Getenv("USER"))
	userHex := make([]byte, hex.EncodedLen(len(user)))
	hex.Encode(userHex, user)
	return userHex
}

func (p *AuthDbusCookieSha1) ProcessData(mesg []byte) ([]byte, error) {
	decodedLen, err := hex.Decode(mesg, mesg)
	if err != nil {
		return nil, err
	}
	mesgTokens := bytes.SplitN(mesg[:decodedLen], []byte(" "), 3)

	file, err := os.Open(os.Getenv("HOME") + "/.dbus-keyrings/" + string(mesgTokens[0]))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileStream := bufio.NewReader(file)

	var cookie []byte
	for {
		line, _, err := fileStream.ReadLine()
		if err == io.EOF {
			return nil, errors.New("SHA1 Cookie not found")
		} else if err != nil {
			return nil, err
		}
		cookieTokens := bytes.SplitN(line, []byte(" "), 3)
		if bytes.Compare(cookieTokens[0], mesgTokens[1]) == 0 {
			cookie = cookieTokens[2]
			break
		}
	}

	challenge := make([]byte, len(mesgTokens[2]))
	if _, err = rand.Read(challenge); err != nil {
		return nil, err
	}

	for temp := challenge; ; {
		if index := bytes.IndexAny(temp, " \t"); index == -1 {
			break
		} else if _, err := rand.Read(temp[index : index+1]); err != nil {
			return nil, err
		} else {
			temp = temp[index:]
		}
	}

	hash := sha1.New()
	if _, err := hash.Write(bytes.Join([][]byte{mesgTokens[2], challenge, cookie}, []byte(":"))); err != nil {
		return nil, err
	}

	resp := bytes.Join([][]byte{challenge, []byte(hex.EncodeToString(hash.Sum(nil)))}, []byte(" "))
	respHex := make([]byte, hex.EncodedLen(len(resp)))
	hex.Encode(respHex, resp)
	return append([]byte("DATA "), respHex...), nil
}

func min(l, r int) int {
	if l < r {
		return l
	}
	return r
}

func (p *Connection) _Authenticate(mech Authenticator) error {
	inStream := bufio.NewReader(p.conn)
	msg := bytes.Join([][]byte{[]byte("AUTH"), mech.Mechanism(), mech.InitialResponse()}, []byte(" "))
	_, err := p.conn.Write(append(msg, "\r\n"...))

	for {
		mesg, _, _ := inStream.ReadLine()

		switch {
		case bytes.HasPrefix(mesg, []byte("DATA")):
			var resp []byte
			resp, err = mech.ProcessData(mesg[min(len("DATA "), len(mesg)):])
			if err != nil {
				p.conn.Write([]byte("CANCEL\r\n"))
			}
			p.conn.Write(append(resp, "\r\n"...))

		case bytes.HasPrefix(mesg, []byte("OK")),
			bytes.HasPrefix(mesg, []byte("AGREE_UNIX_FD")):
			p.conn.Write([]byte("BEGIN\r\n"))
			return nil

		case bytes.HasPrefix(mesg, []byte("REJECTED")):
			if err != nil {
				return err
			}
			return errors.New("Rejected: " + string(mesg[min(len("REJECTED "), len(mesg)):]))

		case bytes.HasPrefix(mesg, []byte("ERROR")):
			return errors.New("Error: " + string(mesg[min(len("ERROR "), len(mesg)):]))

		default:
			p.conn.Write([]byte("ERROR\r\n"))
		}
	}
	return nil
}
