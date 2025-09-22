package tfutils

import "testing"

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "basic snake case",
			input:    "api_ver_1",
			expected: "apiVer1",
		},
		{
			name:     "leading underscore",
			input:    "__lag",
			expected: "lag",
		},
		{
			name:     "single underscore prefix",
			input:    "_members",
			expected: "members",
		},
		{
			name:     "single word",
			input:    "test",
			expected: "test",
		},
		{
			name:     "multiple underscores",
			input:    "hello___world",
			expected: "helloWorld",
		},
		{
			name:     "special name - pool_ipv4",
			input:    "pool_ipv4",
			expected: "poolIPv4",
		},
		{
			name:     "special name - ip_mtu",
			input:    "ip_mtu",
			expected: "ipMTU",
		},
		{
			name:     "special name - vlan_id",
			input:    "vlan_id",
			expected: "vlanID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SnakeToCamel(tt.input)
			if result != tt.expected {
				t.Errorf("SnakeToCamel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "basic camel case",
			input:    "apiVersion1",
			expected: "api_version1",
		},
		{
			name:     "leading underscores",
			input:    "__lag",
			expected: "__lag",
		},
		{
			name:     "mixed case with numbers",
			input:    "_MemberS",
			expected: "_member_s",
		},
		{
			name:     "single word",
			input:    "test",
			expected: "test",
		},
		{
			name:     "special name - poolIPv4",
			input:    "poolIPv4",
			expected: "pool_ipv4",
		},
		{
			name:     "special name - ipMTU",
			input:    "ipMTU",
			expected: "ip_mtu",
		},
		{
			name:     "special name - vlanID",
			input:    "vlanID",
			expected: "vlan_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CamelToSnake(tt.input)
			if result != tt.expected {
				t.Errorf("CamelToSnake(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
