package gen

import (
	"errors"
	"testing"
)

// TestInject checks the injection engine: marker lookup, presence by trimmed
// consecutive lines, block order, indentation and line endings.
//
// Covers: FR-009, EC-011, EC-012, EC-025, EC-030
func TestInject(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		marker  string
		blocks  []string
		want    string
		changed bool
		err     error
	}{
		{
			name:    "indented marker",
			src:     "dependencies = [\n    \"fastapi\",\n    # aicli:dependencies\n]\n",
			marker:  "dependencies",
			blocks:  []string{"\"openai>=3.36\",\n"},
			want:    "dependencies = [\n    \"fastapi\",\n    # aicli:dependencies\n    \"openai>=3.36\",\n]\n",
			changed: true,
		},
		{
			name:    "crlf file",
			src:     "A=1\r\n# aicli:env\r\nB=2\r\n",
			marker:  "env",
			blocks:  []string{"OPENAI_API_KEY=\n"},
			want:    "A=1\r\n# aicli:env\r\nOPENAI_API_KEY=\r\nB=2\r\n",
			changed: true,
		},
		{
			name:    "entry already present",
			src:     "# aicli:env\nX=1\n  OPENAI_API_KEY=  \n",
			marker:  "env",
			blocks:  []string{"OPENAI_API_KEY=\n"},
			want:    "# aicli:env\nX=1\n  OPENAI_API_KEY=  \n",
			changed: false,
		},
		{
			name:    "different version bound inserted",
			src:     "  # aicli:dependencies\n  \"openai>=1.0\",\n",
			marker:  "dependencies",
			blocks:  []string{"\"openai>=3.36\","},
			want:    "  # aicli:dependencies\n  \"openai>=3.36\",\n  \"openai>=1.0\",\n",
			changed: true,
		},
		{
			name:    "several entries in plan order",
			src:     "# aicli:env\nOLD=1\n",
			marker:  "env",
			blocks:  []string{"A=\n", "B=\n", "C=\n"},
			want:    "# aicli:env\nA=\nB=\nC=\nOLD=1\n",
			changed: true,
		},
		{
			name:    "multi-line block with blank and repeated lines",
			src:     "class X:\n    # aicli:capabilities\n    def a(self):\n        return\n\n",
			marker:  "capabilities",
			blocks:  []string{"def b(self):\n    return\n\n"},
			want:    "class X:\n    # aicli:capabilities\n    def b(self):\n        return\n\n    def a(self):\n        return\n\n",
			changed: true,
		},
		{
			name:    "same block twice",
			src:     "# aicli:routers\n",
			marker:  "routers",
			blocks:  []string{"app.include(a)\napp.include(b)\n", "app.include(a)\napp.include(b)\n"},
			want:    "# aicli:routers\napp.include(a)\napp.include(b)\n",
			changed: true,
		},
		{
			name:    "marker on last line without ending",
			src:     "x = 1\r\n# aicli:routers",
			marker:  "routers",
			blocks:  []string{"a\n", "b\n"},
			want:    "x = 1\r\n# aicli:routers\r\na\r\nb\r\n",
			changed: true,
		},
		{
			name:    "only line without ending",
			src:     "# aicli:routers",
			marker:  "routers",
			blocks:  []string{"a"},
			want:    "# aicli:routers\na\n",
			changed: true,
		},
		{
			name:    "marker without ending, nothing inserted",
			src:     "a\n# aicli:routers",
			marker:  "routers",
			blocks:  []string{"a"},
			want:    "a\n# aicli:routers",
			changed: false,
		},
		{
			name:   "no marker",
			src:    "# aicli:other\n# aicli:envs\nx # aicli:env\n",
			marker: "env",
			blocks: []string{"A=\n"},
			err:    ErrNoMarker,
		},
		{
			name:   "repeated marker",
			src:    "# aicli:env\nA=1\n  # aicli:env  \n",
			marker: "env",
			blocks: []string{"B=\n"},
			err:    ErrRepeatedMarker,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := Inject([]byte(tt.src), tt.marker, tt.blocks)
			if tt.err != nil {
				if !errors.Is(err, tt.err) || got != nil || changed {
					t.Fatalf("got (%q, %v, %v), want error %v", got, changed, err, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want || changed != tt.changed {
				t.Fatalf("got (%q, %v), want (%q, %v)", got, changed, tt.want, tt.changed)
			}
		})
	}
}
