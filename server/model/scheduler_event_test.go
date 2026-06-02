package model

import (
	"testing"
	"time"
)

func TestParseSchedulerEventDBTime(t *testing.T) {
	now := time.Now().Round(time.Second)
	cases := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "time value",
			value: now,
		},
		{
			name:  "sqlite datetime string",
			value: "2026-04-27 14:30:45",
		},
		{
			name:  "rfc3339 string",
			value: "2026-04-27T14:30:45+08:00",
		},
		{
			name:  "byte string",
			value: []byte("2026-04-27 14:30:45.123456789+08:00"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := parseSchedulerEventDBTime(tc.value)
			if err != nil {
				t.Fatalf("解析时间失败: %v", err)
			}
			if parsed.IsZero() {
				t.Fatalf("期望解析结果非零时间")
			}
		})
	}
}
