package term

import (
	"bytes"
	"fmt"
	"io"
)

const (
	keySpace  = ' '
	keyReturn = '\n'
	keyEnter  = '\r'
)

type Terminal struct {
	Prompt string
	Status string
	rw     io.ReadWriter
	cmds   map[string]func(*Terminal, ...string) (string, error)
}

func NewTerminal(rw io.ReadWriter, prompt string) *Terminal {
	return &Terminal{
		Prompt: prompt,
		rw:     rw,
		cmds:   make(map[string]func(*Terminal, ...string) (string, error)),
	}
}

func (t *Terminal) AddCmd(key string, operation func(*Terminal, ...string) (string, error)) error {
	if _, ok := t.cmds[key]; ok {
		return fmt.Errorf("The key %s already exist", key)
	}
	t.cmds[key] = operation
	return nil
}

func (t *Terminal) Run() {
	close := make(chan struct{}, 1)
	t.AddCmd("exit", func(*Terminal, ...string) (string, error) {
		close <- struct{}{}
		return "client closed", nil
	})
	for {
		select {
		case <-close:
			return
		default:
			status := ""
			if t.Status != "" {
				status = " (" + t.Status + ")"
			}
			t.rw.Write([]byte("\033[0;33m" + t.Prompt + status + " > \033[0m"))
			inBuf := make([]byte, 256)
			t.rw.Read(inBuf)
			cmds := ParseLine(inBuf)
			if len(cmds) > 0 {
				t.OperateCmd(cmds[0], cmds[1:]...)
			}
		}
	}
}

func (t *Terminal) WriteErrorMessage(err error) {
	t.writeMessage("\033[0;31m", "ERROR", err.Error())
}

func (t *Terminal) WriteInfoMessage(msg string) {
	t.writeMessage("\033[0;36m", "INFO", msg)
}

func (t *Terminal) writeMessage(color, title, msg string) {
	title = color + t.Prompt + " (" + title + ")" + " > \033[0m"
	t.rw.Write([]byte(title))
	t.rw.Write([]byte(msg + "\r\n"))
}

func (t *Terminal) OperateCmd(cmd string, args ...string) (err error) {
	fn, ok := t.cmds[cmd]
	if !ok {
		err = fmt.Errorf("The command does not exist, please try again")
		t.WriteErrorMessage(err)
		return
	}
	msg, err := fn(t, args...)
	if err != nil {
		t.WriteErrorMessage(err)
		return
	}
	t.WriteInfoMessage(msg)
	return
}

func ParseLine(line []byte) []string {
	var cmds []string
	var tempBuf bytes.Buffer
	for _, c := range line {
		switch c {
		case keySpace, keyEnter, keyReturn:
			if tempBuf.Len() > 0 {
				cmds = append(cmds, tempBuf.String())
				tempBuf.Reset()
			}
		default:
			tempBuf.WriteByte(c)
		}
		if c == keyEnter || c == keyReturn {
			return cmds
		}
	}
	return cmds
}
