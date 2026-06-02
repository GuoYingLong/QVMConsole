package service

import "testing"

func TestDetectVMOSType(t *testing.T) {
	testCases := []struct {
		name         string
		templateName string
		xmlStr       string
		expected     string
	}{
		{
			name:         "detect fnos from template name fallback",
			templateName: "fnos-template",
			expected:     "fnos",
		},
		{
			name:     "detect windows from xml",
			xmlStr:   "<domain><os firmware='efi'></os><features><hyperv/></features></domain>",
			expected: "windows",
		},
		{
			name:     "fallback to linux",
			xmlStr:   "<domain><os><type>hvm</type></os></domain>",
			expected: "linux",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := detectVMOSType(testCase.templateName, testCase.xmlStr)
			if got != testCase.expected {
				t.Fatalf("expected os type %q, got %q", testCase.expected, got)
			}
		})
	}
}

func TestParseHostNeighborIPsByMAC(t *testing.T) {
	text := `10.200.3.1 dev vpcsw6 lladdr aa:bb:cc:dd:ee:ff REACHABLE
10.200.3.134 dev vpcsw6 lladdr 52:54:00:33:bf:81 STALE
10.200.4.20 dev vpcsw7 lladdr 52:54:00:33:bf:81 REACHABLE`

	got := parseHostNeighborIPsByMAC(text, "52:54:00:33:BF:81", "10.200.3.0/24")
	if len(got) != 1 || got[0] != "10.200.3.134" {
		t.Fatalf("expected VPC neighbor IP 10.200.3.134, got %#v", got)
	}
}
