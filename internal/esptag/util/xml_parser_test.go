package util

import (
	"testing"
)

func TestFindTagParentAndPath(t *testing.T) {
	tests := []struct {
		name        string
		xmlInput    string
		targetTag   string
		wantParent  string
		wantSubPath []string
		wantErr     bool
	}{
		{
			name: "XML válido com tag simples",
			xmlInput: `<?xml version="1.0"?>
				<root>
					<parent>
						<xpto>value</xpto>
						<target>value</target>
					</parent>
				</root>`,
			targetTag:   "target",
			wantParent:  "parent",
			wantSubPath: []string{"root", "parent", "target"},
			wantErr:     false,
		},
		{
			name: "XML com namespace",
			xmlInput: `<?xml version="1.0"?>
				<ns:root xmlns:ns="http://example.com">
					<ns:parent>
						<ns:target>value</ns:target>
					</ns:parent>
				</ns:root>`,
			targetTag:   "target",
			wantParent:  "parent",
			wantSubPath: []string{"root", "parent", "target"},
			wantErr:     false,
		},
		{
			name: "Tag não encontrada",
			xmlInput: `<?xml version="1.0"?>
				<root>
					<parent>
						<other>value</other>
					</parent>
				</root>`,
			targetTag: "target",
			wantErr:   true,
		},
		{
			name:      "XML inválido",
			xmlInput:  "<root><unclosed>",
			targetTag: "target",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, subPath, err := FindTagParentAndPath(tt.xmlInput, tt.targetTag)

			if (err != nil) != tt.wantErr {
				t.Errorf("findTagParentAndPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if parent != tt.wantParent {
					t.Errorf("findTagParentAndPath() parent = %v, want %v", parent, tt.wantParent)
				}

				if len(subPath) != len(tt.wantSubPath) {
					t.Errorf("findTagParentAndPath() subPath length = %v, want %v", len(subPath), len(tt.wantSubPath))
				} else {
					for i := range subPath {
						if subPath[i] != tt.wantSubPath[i] {
							t.Errorf("findTagParentAndPath() subPath[%d] = %v, want %v", i, subPath[i], tt.wantSubPath[i])
						}
					}
				}
			}
		})
	}
}
