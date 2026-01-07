package cmd

import "testing"

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"goodname", false},
		{"good-name", false},
		{"-bad", true},
		{"bad-", true},
		{"bad name", true},
		{"BAD", true},
		{"", true},
	}

	for _, tc := range cases {
		err := validateName(tc.name)
		if tc.wantErr && err == nil {
			t.Fatalf("expected error for %q", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.name, err)
		}
	}
}

func TestNormalizeTLDs(t *testing.T) {
	input := []string{".com", " io ", "com", "dev,app"}
	result := normalizeTLDs(input)
	if len(result) != 4 {
		t.Fatalf("expected 4 tlds, got %d", len(result))
	}
}
