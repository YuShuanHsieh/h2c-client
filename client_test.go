package main

import (
	"bytes"
	"testing"

	"github.com/YuShuanHsieh/h2c-client/term"
)

func TestSettingCmd(t *testing.T) {
	var buf bytes.Buffer
	client := newClient()
	term := term.NewTerminal(&buf, "test")

	term.AddCmd("settings", settingsCmd(client))

	tests := []struct {
		test     []string
		expected bool
	}{
		{
			test:     []string{"push=123", "maxStream"},
			expected: false,
		},
		{
			test:     []string{"push=123", "maxStream=2345"},
			expected: true,
		},
		{
			test:     []string{"push"},
			expected: false,
		},
		{
			test:     []string{},
			expected: true,
		},
	}

	for i, test := range tests {
		err := term.OperateCmd("settings", test.test...)
		if (err != nil && test.expected == true) || (err == nil && test.expected == false) {
			t.Errorf("Test case [%d] is failed. Expected %v but get %v", i, test.expected, err)
		}
	}

	if client.settings.enablePush.value != 123 {
		t.Error("setting push failed")
	}
	if client.settings.maxConcurrentStreams.value != 2345 {
		t.Error("setting maxStream failed")
	}
}
