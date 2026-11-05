package sanitize

import (
	"testing"
)

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		// 2026-11-05 09:38 +03: Kuwait format variants moved to
		// TestSanitizePhoneKuwaitVariants (number from KWTSMS_TEST_NUMBER), int-testnumber.
		// local trunk prefix stripped: +966 055-XXX-XXXX → 96655XXXXXXX
		{"+966 055 123 4567", "966551234567", false},
		{"abc", "", true},
		{"", "", true},
		{"   ", "", true},
	}
	for _, tt := range tests {
		got, err := SanitizePhone(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("SanitizePhone(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("SanitizePhone(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// 2026-11-05 09:38 +03: Kuwait cases moved here from TestSanitizePhone; the
// number comes from KWTSMS_TEST_NUMBER (testNumber, skips when unset) and its
// format variants are built at runtime. Subtest names and failure messages
// never contain the number. int-testnumber.
func TestSanitizePhoneKuwaitVariants(t *testing.T) {
	n := testNumber(t)
	cc, a, b := splitKW(n)
	tests := []struct {
		name  string
		input string
	}{
		{"plus prefix", "+" + n},
		{"00 prefix", "00" + n},
		{"spaces", cc + " " + a + " " + b},
		{"dashes", cc + "-" + a + "-" + b},
		{"parens and dash", "(" + cc + ") " + a + "-" + b},
		{"Arabic-Indic digits", toArabicIndic(n)},
		{"Persian digits", toPersian(n)},
		{"already clean", n},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizePhone(tt.input)
			if err != nil {
				t.Errorf("SanitizePhone(%s) returned an error", tt.name)
				return
			}
			if got != n {
				t.Errorf("SanitizePhone(%s) did not return the test number", tt.name)
			}
		})
	}
}

// 2026-11-05 09:38 +03: the Kuwait number comes from KWTSMS_TEST_NUMBER (skips
// when unset); the second number is a Saudi sample instead of a Kuwait one.
// Error-only cases stay in TestSanitizePhonesInvalid so they run without the
// env var. Failure messages never contain the numbers. int-testnumber.
func TestSanitizePhones(t *testing.T) {
	n := testNumber(t)
	cc, a, b := splitKW(n)
	const other = "966551234567"
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"comma", n + "," + other, []string{n, other}},
		// spaces within a token are stripped per-number, not treated as delimiters
		{"spaces within token", cc + " " + a + " " + b, []string{n}},
		{"plus prefixes", "+" + n + ",+" + other, []string{n, other}},
		{"newline", n + "\n" + other, []string{n, other}},
		{"comma and space", n + ", " + other, []string{n, other}},
		// deduplication
		{"dedupe", n + "," + n + "," + other, []string{n, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizePhones(tt.input)
			if err != nil {
				t.Errorf("SanitizePhones(%s) returned an error", tt.name)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("SanitizePhones(%s) returned %d numbers, want %d", tt.name, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SanitizePhones(%s)[%d] is not the expected number", tt.name, i)
				}
			}
		})
	}
}

// 2026-11-05 09:38 +03: split from TestSanitizePhones so it runs without
// KWTSMS_TEST_NUMBER, int-testnumber.
func TestSanitizePhonesInvalid(t *testing.T) {
	for _, in := range []string{"", "abc"} {
		if _, err := SanitizePhones(in); err == nil {
			t.Errorf("SanitizePhones(%q) error = nil, want error", in)
		}
	}
}

func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"Hello world", "Hello world", false},
		{"<b>Hello</b>", "Hello", false},
		{"Hello<script>alert(1)</script>world", "Helloalert(1)world", false},
		{"Hello&#x3C;world", "Helloworld", false},
		{"Hello\x00world", "Helloworld", false},
		{"Hello\nworld", "Hello\nworld", false},
		{"Hello\tworld", "Hello\tworld", false},
		// SQL meta stripped
		{"Hello; DROP TABLE", "Hello DROP TABLE", false},
		{"Hello--comment", "Hellocomment", false},
		// Zero-width space stripped
		{"Hello\u200Bworld", "Helloworld", false},
		// BOM stripped
		{"\uFEFFHello", "Hello", false},
		// Empty after sanitization
		{"<script></script>", "", true},
		{"", "", true},
		{"   ", "", true},
	}
	for _, tt := range tests {
		got, err := SanitizeMessage(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("SanitizeMessage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("SanitizeMessage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeSenderID(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"MY-SENDER", "MY-SENDER", false},
		{"App.Name", "App.Name", false},
		{"My Brand", "My Brand", false},
		{"KWT-SMS", "KWT-SMS", false},
		{"ABC123", "ABC123", false},
		{"  MY-SENDER  ", "MY-SENDER", false},
		// Invalid characters
		{"<script>", "", true},
		{"sender!", "", true},
		{"sender@id", "", true},
		// Too long (>11 chars)
		{"TOOLONGNAME12", "", true},
		// Empty
		{"", "", true},
		{"   ", "", true},
	}
	for _, tt := range tests {
		got, err := SanitizeSenderID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("SanitizeSenderID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("SanitizeSenderID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeConfigValue(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"myvalue", "myvalue", false},
		{"  myvalue  ", "myvalue", false},
		{`"myvalue"`, "myvalue", false},
		{"'myvalue'", "myvalue", false},
		{`"  myvalue  "`, "myvalue", false},
		// Empty
		{"", "", true},
		{"   ", "", true},
		{`""`, "", true},
		{"''", "", true},
	}
	for _, tt := range tests {
		got, err := SanitizeConfigValue(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("SanitizeConfigValue(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("SanitizeConfigValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
