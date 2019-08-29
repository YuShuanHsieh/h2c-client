package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/YuShuanHsieh/h2c-client/client"
	"github.com/YuShuanHsieh/h2c-client/term"
)

func closeCmd(client client.Client) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (string, error) {
		if client.Close() == nil {
			t.Status = ""
			return "connetion closed", nil
		}
		return "", fmt.Errorf("no connection")
	}
}

func sendRequestCmd(client client.Client) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (msg string, err error) {
		if !client.IsAvailable() {
			err = fmt.Errorf("connection has been closed")
			return
		}
		// format <method or ping> </path>
		if len(args) == 0 {
			return "", fmt.Errorf("no methods")
		}
		switch args[0] {
		case "PING":
			if err = client.PING(); err != nil {
				err = fmt.Errorf("PING failed %v", err)
				return
			}
		case "GET":
			if len(args) < 2 {
				err = fmt.Errorf("GET method must have a PATH")
				return
			}
			var resp *http.Response
			resp, err = client.GET(args[1])
			if err != nil {
				err = fmt.Errorf("send GET request failed %v", err)
				return
			}
			respData, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			t.WriteInfoMessage(fmt.Sprintf("status sode: %d", resp.StatusCode))
			t.WriteInfoMessage(fmt.Sprintf("data: %s", respData))
		default:
			return "", fmt.Errorf("Invalid method")
		}
		return "send request success", nil
	}
}

func settingsCmd(client client.Client) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (msg string, err error) {
		var value int
		var changed bool
		// format => maxStream=250
		for _, v := range args {
			results := strings.Split(v, "=")
			if len(results) < 2 || results[1] == "" {
				return "", fmt.Errorf("The cmd %s is ivalid format(setting=value)", v)
			}
			value, err = strconv.Atoi(results[1])
			if err != nil {
				return "", fmt.Errorf("Invalid value %s", results[1])
			}
			if changed, err = client.UpdateSetting(results[0], value); err != nil {
				err = fmt.Errorf("Update setting failed %v", err)
				return
			}
			if changed {
				t.WriteInfoMessage(fmt.Sprintf("setting %s was updated to %s", results[0], results[1]))
			}
		}
		client.SETTINGS()

		updated := fmt.Sprintf(
			"Enable Push: %d | Max Concurrent Streams: %d | Init Window Size: %d | Max Frame Size: %d",
			client.GetSetting("push"),
			client.GetSetting("maxStream"),
			client.GetSetting("windowSize"),
			client.GetSetting("frameSize"),
		)
		return updated, nil
	}
}

func connectCmd(client client.Client) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (msg string, err error) {
		if len(args) < 1 {
			return "", fmt.Errorf("missing host")
		}
		host := args[0]
		if h, _, err := net.SplitHostPort(args[0]); err != nil {
			host = net.JoinHostPort(h, "80")
		}
		if err = client.Connect(host); err != nil {
			return
		}
		t.Status = "connected to " + host
		return "create a connection to server", nil
	}
}

func main() {
	rw := struct {
		io.Reader
		io.Writer
	}{
		os.Stdin,
		os.Stdout,
	}
	client := client.NewClient()
	term := term.NewTerminal(rw, "h2c")
	term.AddCmd("settings", settingsCmd(client))
	term.AddCmd("connect", connectCmd(client))
	term.AddCmd("send", sendRequestCmd(client))
	term.AddCmd("close", closeCmd(client))

	term.Run()
}
