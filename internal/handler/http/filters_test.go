package http

import (
	"testing"
)

func TestParseStringFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *string
	}{
		{"valid string", "  hello  ", strPtr("hello")},
		{"empty string", "", nil},
		{"whitespace only", "   ", nil},
		{"max length 255", string(make([]byte, 255)), strPtr(string(make([]byte, 255)))},
		{"over max length", string(make([]byte, 256)), nil},
		{"exact max length", "a", strPtr("a")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseStringFilter(tc.input)
			if (result == nil) != (tc.expected == nil) {
				t.Fatalf("parseStringFilter(%q) = %v, want %v", tc.input, result, tc.expected)
			}
			if result != nil && *result != *tc.expected {
				t.Fatalf("parseStringFilter(%q) = %q, want %q", tc.input, *result, *tc.expected)
			}
		})
	}
}

func TestParseTerminalFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectNil   bool
		expectError bool
		expectedVal *int
	}{
		{"valid positive", "42", false, false, intPtr(42)},
		{"valid zero", "0", false, false, intPtr(0)},
		{"max int32", "2147483647", false, false, intPtr(2147483647)},
		{"empty string", "", true, false, nil},
		{"whitespace only", "   ", true, false, nil},
		{"negative", "-1", true, true, nil},
		{"over max int32", "2147483648", true, true, nil},
		{"non-numeric", "abc", true, true, nil},
		{"float", "1.5", true, true, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseTerminalFilter(tc.input)
			if tc.expectError {
				if err == nil {
					t.Fatalf("parseTerminalFilter(%q) expected error, got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Fatalf("parseTerminalFilter(%q) unexpected error: %v", tc.input, err)
				}
			}
			if tc.expectNil {
				if result != nil {
					t.Fatalf("parseTerminalFilter(%q) expected nil, got %v", tc.input, result)
				}
			} else if result == nil {
				t.Fatalf("parseTerminalFilter(%q) expected value, got nil", tc.input)
			} else if *result != *tc.expectedVal {
				t.Fatalf("parseTerminalFilter(%q) = %d, want %d", tc.input, *result, *tc.expectedVal)
			}
		})
	}
}

func TestParseStatusFilter(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectError    bool
		expectedStatus *int16
		expectedMin    *int16
	}{
		{"special failed", "failed", false, nil, int16Ptr(40)},
		{"special gagal", "gagal", false, nil, int16Ptr(40)},
		{"special FAILED case", "FAILED", true, nil, nil},
		{"special GAGAL case", "GAGAL", true, nil, nil},
		{"exact 0", "0", false, int16Ptr(0), nil},
		{"exact 20", "20", false, int16Ptr(20), nil},
		{"exact 40", "40", false, int16Ptr(40), nil},
		{"exact 32767", "32767", false, int16Ptr(32767), nil},
		{"empty string", "", false, nil, nil},
		{"whitespace only", "   ", false, nil, nil},
		{"negative", "-1", true, nil, nil},
		{"over int16 max", "32768", true, nil, nil},
		{"over int16 max 99999", "99999", true, nil, nil},
		{"non-numeric", "abc", true, nil, nil},
		{"float", "1.5", true, nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, statusMin, err := parseStatusFilter(tc.input)
			if tc.expectError {
				if err == nil {
					t.Fatalf("parseStatusFilter(%q) expected error, got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Fatalf("parseStatusFilter(%q) unexpected error: %v", tc.input, err)
				}
			}
			if (status == nil) != (tc.expectedStatus == nil) {
				t.Fatalf("parseStatusFilter(%q) status = %v, want %v", tc.input, status, tc.expectedStatus)
			}
			if status != nil && *status != *tc.expectedStatus {
				t.Fatalf("parseStatusFilter(%q) status = %d, want %d", tc.input, *status, *tc.expectedStatus)
			}
			if (statusMin == nil) != (tc.expectedMin == nil) {
				t.Fatalf("parseStatusFilter(%q) statusMin = %v, want %v", tc.input, statusMin, tc.expectedMin)
			}
			if statusMin != nil && *statusMin != *tc.expectedMin {
				t.Fatalf("parseStatusFilter(%q) statusMin = %d, want %d", tc.input, *statusMin, *tc.expectedMin)
			}
		})
	}
}

func TestParseBoolFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedVal *bool
	}{
		{"true string", "true", false, boolPtr(true)},
		{"true number", "1", false, boolPtr(true)},
		{"false string", "false", false, boolPtr(false)},
		{"false number", "0", false, boolPtr(false)},
		{"empty string", "", false, nil},
		{"whitespace only", "   ", false, nil},
		{"yes", "yes", true, nil},
		{"no", "no", true, nil},
		{"null string", "null", true, nil},
		{"random", "abc", true, nil},
		{"number 2", "2", true, nil},
		{"truee typo", "truee", true, nil},
		{"falsee typo", "falsee", true, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseBoolFilter(tc.input)
			if tc.expectError {
				if err == nil {
					t.Fatalf("parseBoolFilter(%q) expected error, got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Fatalf("parseBoolFilter(%q) unexpected error: %v", tc.input, err)
				}
			}
			if (result == nil) != (tc.expectedVal == nil) {
				t.Fatalf("parseBoolFilter(%q) = %v, want %v", tc.input, result, tc.expectedVal)
			}
			if result != nil && *result != *tc.expectedVal {
				t.Fatalf("parseBoolFilter(%q) = %v, want %v", tc.input, *result, *tc.expectedVal)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func int16Ptr(i int16) *int16 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
