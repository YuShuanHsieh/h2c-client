package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"

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

type h2cClient struct {
	conn      net.Conn
	http2Conn *http2.ClientConn
	settings  settings
	url       url.URL
}

func NewClient() Client {
	return &h2cClient{
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

func (c *h2cClient) Close() error {
	if c.http2Conn != nil {
		return c.http2Conn.Shutdown(context.Background())
	}
	return nil
}

func addH2cUpgrade(req *http.Request, h2cSettings settings) {
	settings := base64.RawURLEncoding.EncodeToString(h2cSettings.ToBytes())
	req.Header.Add("Upgrade", "h2c")
	req.Header.Add("Connection", "HTTP2-Settings")
	req.Header.Add("HTTP2-Settings", settings)
}

func (c *h2cClient) SendUpgradeRequest() (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, c.url.String(), nil)
	addH2cUpgrade(req, c.settings)

	req.Write(c.conn)
	buf := bufio.NewReader(c.conn)

	return http.ReadResponse(buf, req)
}

func (c *h2cClient) IsAvailable() bool {
	if c.http2Conn == nil {
		return false
	}
	return c.http2Conn.CanTakeNewRequest()
}

func (c *h2cClient) PING() error {
	return c.http2Conn.Ping(context.Background())
}

func (c *h2cClient) GET(path string) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, c.url.String()+path, nil)
	return c.http2Conn.RoundTrip(req)
}

func (c *h2cClient) UpdateSetting(key string, value int) (changed bool, err error) {
	var target *http2.Setting
	switch key {
	case "push":
		if value > 0 {
			c.settings.enablePush.Val = 1
		} else {
			c.settings.enablePush.Val = 0
		}
		return true, nil
	case "maxStream":
		target = &c.settings.maxConcurrentStreams
	case "windowSize":
		target = &c.settings.initWindowSize
	case "frameSize":
		target = &c.settings.maxFrameSize
	default:
		return false, fmt.Errorf("invalid setting %s", key)
	}
	return applySetting(value, target), nil
}

func (c *h2cClient) SETTINGS() error {
	if !c.IsAvailable() {
		return fmt.Errorf("The connection is not available")
	}
	framer := http2.NewFramer(c.conn, c.conn)
	return framer.WriteSettings(c.settings.enablePush, c.settings.initWindowSize, c.settings.maxConcurrentStreams, c.settings.maxFrameSize)
}

func applySetting(cfg int, setting *http2.Setting) bool {
	if cfg > 0 {
		setting.Val = uint32(cfg)
		return true
	}
	return false
}

func (c *h2cClient) GetSetting(key string) uint32 {
	switch key {
	case "push":
		return c.settings.enablePush.Val
	case "maxStream":
		return c.settings.maxConcurrentStreams.Val
	case "windowSize":
		return c.settings.initWindowSize.Val
	case "frameSize":
		return c.settings.maxFrameSize.Val
	}
	return 0
}

func (c *h2cClient) Connect(host string) error {
	c.Close()
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return fmt.Errorf("dial to server [%s] failed %v", host, err)
	}
	c.conn = conn

	c.url = url.URL{
		Scheme: "http",
		Host:   host,
	}

	resp, err := c.SendUpgradeRequest()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("The server does not support h2c")
	}
	tr := &http2.Transport{AllowHTTP: true}

	c.http2Conn, err = tr.NewClientConn(c.conn)
	if err != nil {
		return fmt.Errorf("init HTTP/2 connection failed %v", err)
	}
	return nil
}
