package client

import (
	"net/http"
)

type Client interface {
	SendUpgradeRequest() (*http.Response, error)
	Close() error
	IsAvailable() bool
	GetSetting(key string) uint32
	UpdateSetting(key string, value int) (bool, error)
	PING() error
	GET(string) (*http.Response, error)
	SETTINGS() error
	Connect(string) error
}
