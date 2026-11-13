package proxy

import "testing"

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"valid http URL", "http://monolith:8080", false},
		{"valid https URL with path", "https://example.com/api", false},
		{"missing scheme", "monolith:8080", true},
		{"missing host", "http://", true},
		{"opaque URL without host", "http:monolith", true},
		{"empty string", "", true},
		{"unparseable URL", "http://exa mple.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := parseTarget(tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTarget(%q) error = %v, wantErr %v", tt.target, err, tt.wantErr)
			}
			if err == nil && (u.Scheme == "" || u.Host == "") {
				t.Errorf("parseTarget(%q) accepted URL with scheme=%q host=%q", tt.target, u.Scheme, u.Host)
			}
		})
	}
}
