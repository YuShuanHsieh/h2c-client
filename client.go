package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/YuShuanHsieh/h2c-client/term"
	"golang.org/x/net/http2"
)

type settings struct {
	enablePush           http2.Setting
	maxConcurrentStreams http2.Setting
	initWindowSize       http2.Setting
	maxFrameSize         http2.Setting
}

func (s *settings) ToBytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, s)
	return buf.Bytes()
}

type H2cClient struct {
	conn      net.Conn
	http2Conn *http2.ClientConn
	settings  settings
	url       url.URL
	term      *term.Terminal
}

func newClient() *H2cClient {
	return &H2cClient{
		settings: settings{
			enablePush: http2.Setting{
				ID:  http2.SettingEnablePush,
				Val: 0,
			},
			maxConcurrentStreams: http2.Setting{
				ID:  http2.SettingMaxConcurrentStreams,
				Val: 250,
			},
			initWindowSize: http2.Setting{
				ID:  http2.SettingInitialWindowSize,
				Val: 65535,
			},
			maxFrameSize: http2.Setting{
				ID:  http2.SettingMaxFrameSize,
				Val: 16384,
			},
		},
	}
}

func applySetting(cfg int, setting *http2.Setting) bool {
	if cfg > 0 {
		setting.Val = uint32(cfg)
		return true
	}
	return false
}

func addH2cUpgrade(req *http.Request, h2cSettings settings) {
	settings := base64.RawURLEncoding.EncodeToString(h2cSettings.ToBytes())
	req.Header.Add("Upgrade", "h2c")
	req.Header.Add("Connection", "HTTP2-Settings")
	req.Header.Add("HTTP2-Settings", settings)
}

func (c *H2cClient) sendUpgradeRequest() (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, c.url.String(), nil)
	addH2cUpgrade(req, c.settings)

	req.Write(c.conn)
	buf := bufio.NewReader(c.conn)

	return http.ReadResponse(buf, req)
}

func closeCmd(client *H2cClient) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (string, error) {
		if client.http2Conn != nil {
			client.http2Conn.Close()
			client.term.Status = ""
			return "connetion closed", nil
		}
		return "", fmt.Errorf("no connection")
	}
}

func sendRequestCmd(client *H2cClient) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (string, error) {
		if client.http2Conn == nil {
			return "", fmt.Errorf("please use connect cmd to connect to server")
		}
		if !client.http2Conn.CanTakeNewRequest() {
			t.Status = ""
			return "", fmt.Errorf("connection has been closed")
		}
		// format <method or ping> </path>
		if len(args) == 0 {
			return "", fmt.Errorf("no methods")
		}
		switch args[0] {
		case "PING":
			if err := client.http2Conn.Ping(context.Background()); err != nil {
				return "", fmt.Errorf("PING failed %v", err)
			}
		case "GET":
			if len(args) < 2 {
				return "", fmt.Errorf("GET method must have a PATH")
			}
			req, _ := http.NewRequest(http.MethodGet, client.url.String()+args[1], nil)
			resp, err := client.http2Conn.RoundTrip(req)
			if err != nil {
				return "", fmt.Errorf("send GET request failed %v", err)
			}
			t.WriteInfoMessage(fmt.Sprintf("status sode: %d", resp.StatusCode))
			respData, _ := ioutil.ReadAll(resp.Body)
			t.WriteInfoMessage(fmt.Sprintf("data: %s", respData))
		default:
			return "", fmt.Errorf("Invalid method")
		}
		return "send request success", nil
	}
}

func settingsCmd(client *H2cClient) func(*term.Terminal, ...string) (string, error) {
	m := map[string]*http2.Setting{
		"push":       &client.settings.enablePush,
		"maxStream":  &client.settings.maxConcurrentStreams,
		"windowSize": &client.settings.initWindowSize,
		"frameSize":  &client.settings.maxFrameSize,
	}
	return func(t *term.Terminal, args ...string) (string, error) {
		// format => maxStream=250
		for _, v := range args {
			results := strings.Split(v, "=")
			if len(results) < 2 || results[1] == "" {
				return "", fmt.Errorf("The cmd %s is ivalid format(setting=value)", v)
			}
			value, err := strconv.Atoi(results[1])
			if err != nil {
				return "", fmt.Errorf("Invalid value %s", results[1])
			}
			if setting, ok := m[results[0]]; !ok {
				return "", fmt.Errorf("Invalid setting option %s", results[0])
			} else {
				if applySetting(value, setting) {
					t.WriteInfoMessage(fmt.Sprintf("setting %s was updated to %s", results[0], results[1]))
				}
			}
		}
		push := false
		if client.settings.enablePush.Val > 0 {
			push = true
		}

		if client.http2Conn != nil && client.http2Conn.CanTakeNewRequest() {
			framer := http2.NewFramer(client.conn, client.conn)
			framer.WriteSettings(client.settings.enablePush, client.settings.initWindowSize, client.settings.maxConcurrentStreams, client.settings.maxFrameSize)
		}

		updated := fmt.Sprintf(
			"Enable Push: %t | Max Concurrent Streams: %d | Init Window Size: %d | Max Frame Size: %d",
			push,
			client.settings.maxConcurrentStreams.Val,
			client.settings.initWindowSize.Val,
			client.settings.maxFrameSize.Val,
		)
		return updated, nil
	}
}

func connectCmd(client *H2cClient) func(*term.Terminal, ...string) (string, error) {
	return func(t *term.Terminal, args ...string) (string, error) {
		if len(args) < 1 {
			return "", fmt.Errorf("missing host")
		}
		host := args[0]
		if h, _, err := net.SplitHostPort(args[0]); err != nil {
			host = net.JoinHostPort(h, "80")
		}

		// close previous connection
		if client.http2Conn != nil && client.http2Conn.CanTakeNewRequest() {
			client.http2Conn.Close()
		}

		conn, err := net.Dial("tcp", host)
		if err != nil {
			return "", fmt.Errorf("dial to server [%s] failed %v", host, err)
		}
		client.conn = conn

		client.url = url.URL{
			Scheme: "http",
			Host:   host,
		}

		resp, err := client.sendUpgradeRequest()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			return "", fmt.Errorf("The server does not support h2c")
		}
		tr := &http2.Transport{AllowHTTP: true}

		client.http2Conn, err = tr.NewClientConn(client.conn)
		if err != nil {
			return "", fmt.Errorf("init HTTP/2 connection failed %v", err)
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
	client := newClient()
	term := term.NewTerminal(rw, "h2c")
	term.AddCmd("settings", settingsCmd(client))
	term.AddCmd("connect", connectCmd(client))
	term.AddCmd("send", sendRequestCmd(client))
	term.AddCmd("close", closeCmd(client))

	client.term = term

	term.Run()
}
