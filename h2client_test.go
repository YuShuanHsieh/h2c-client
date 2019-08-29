package main

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/YuShuanHsieh/h2c-client/client/mocks"
	"github.com/YuShuanHsieh/h2c-client/term"
)

func TestSettingCmd(t *testing.T) {
	var buf bytes.Buffer

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockClient(ctrl)

	mockClient.EXPECT().UpdateSetting(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	mockClient.EXPECT().SETTINGS().Return(nil).AnyTimes()
	mockClient.EXPECT().GetSetting(gomock.Any()).Return(uint32(0)).AnyTimes()

	term := term.NewTerminal(&buf, "test")

	term.AddCmd("settings", settingsCmd(mockClient))

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
}
