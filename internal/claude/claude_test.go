package claude

import "testing"

func TestExtractResult(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid envelope",
			input: `{"type":"result","subtype":"success","result":"hello world","duration_ms":100}`,
			want:  "hello world",
		},
		{
			name:  "envelope with escaped JSON in result",
			input: `{"type":"result","result":"{\"source_extensions\":[\".go\"]}"}`,
			want:  `{"source_extensions":[".go"]}`,
		},
		{
			name:    "not JSON",
			input:   "plain text output",
			wantErr: true,
		},
		{
			name:    "empty result field",
			input:   `{"type":"result","result":""}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractResult(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractResult() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with json fence",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "with plain fence",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "no fence",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "with leading whitespace and newlines",
			input: "\n\n```json\n{\"a\": 1}\n```\n",
			want:  `{"a": 1}`,
		},
		{
			name:  "multiline content in fence",
			input: "```json\n{\n  \"source_extensions\": [\".go\", \".py\"],\n  \"non_source_extensions\": [\".md\"]\n}\n```",
			want:  "{\n  \"source_extensions\": [\".go\", \".py\"],\n  \"non_source_extensions\": [\".md\"]\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFences(tt.input)
			if got != tt.want {
				t.Errorf("stripCodeFences() = %q, want %q", got, tt.want)
			}
		})
	}
}
