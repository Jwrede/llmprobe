package provider

import "testing"

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid object", `{"key": "value"}`, false},
		{"valid array", `[1, 2, 3]`, false},
		{"valid nested", `{"a": {"b": [1, 2]}}`, false},
		{"invalid", `{"key": broken}`, true},
		{"partial", `{"key":`, true},
		{"empty string", ``, true},
		{"plain text", `hello world`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
