package nubank

import "testing"

func TestNubankParserIMPL_Supports(t *testing.T) {

	testCases := []struct {
		name            string
		header          []string
		expectedColumns map[string]int
	}{
		{
			name:   "supports Nubank header",
			header: []string{"Data", "Descrição", "Valor", "Identificador"},
			expectedColumns: map[string]int{
				"date":        0,
				"description": 1,
				"amount":      2,
				"external_id": 3,
			},
		},
		{
			name:   "supports Nubank header with date in the second column",
			header: []string{"Descrição", "Data", "Valor", "Identificador"},
			expectedColumns: map[string]int{
				"date":        1,
				"description": 0,
				"amount":      2,
				"external_id": 3,
			},
		},
		{
			name:   "supports Nubank header date in the third column",
			header: []string{"Descrição", "Valor", "Data", "Identificador"},
			expectedColumns: map[string]int{
				"date":        2,
				"description": 0,
				"amount":      1,
				"external_id": 3,
			},
		},
		{
			name:   "supports Nubank header date in the forth column",
			header: []string{"Descrição", "Valor", "Identificador", "Data"},
			expectedColumns: map[string]int{
				"date":        3,
				"description": 0,
				"amount":      1,
				"external_id": 2,
			},
		},

		{
			name:   "supports Nubank header description in the third column",
			header: []string{"Valor", "Identificador", "Descrição", "Data"},
			expectedColumns: map[string]int{
				"date":        3,
				"description": 2,
				"amount":      0,
				"external_id": 1,
			},
		},

		{
			name:   "supports Nubank header description in the forth column",
			header: []string{"Valor", "Identificador", "Data", "Descrição"},
			expectedColumns: map[string]int{
				"date":        2,
				"description": 3,
				"amount":      0,
				"external_id": 1,
			},
		},

		{
			name:   "supports Nubank header amount in the forth column",
			header: []string{"Identificador", "Data", "Descrição", "Valor"},
			expectedColumns: map[string]int{
				"date":        1,
				"description": 2,
				"amount":      3,
				"external_id": 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := NewNubankParserIMPL(&tc.header)
			if err != nil {
				t.Fatalf("Failed to create NubankParserIMPL: %v", err)
			}

			if !parser.Supports() {
				t.Errorf("Expected Supports() to return true, but got false")
			}

			if len(parser.columns) != len(tc.expectedColumns) {
				t.Errorf("Expected columns length to be %d, but got %d", len(tc.expectedColumns), len(parser.columns))
			}

			for key, expectedIndex := range tc.expectedColumns {
				if index, exists := parser.columns[key]; !exists || index != expectedIndex {
					t.Errorf("Expected column '%s' to have index %d, but got %d", key, expectedIndex, index)
				}
			}
		})
	}

}

func TestNubankParserIMPL_DoesNotSupportInvalidHeaders(t *testing.T) {
	tests := []struct {
		name   string
		header []string
	}{
		{name: "wrong length", header: []string{"Data", "Descrição", "Valor"}},
		{name: "unknown column", header: []string{"Data", "Descrição", "Valor", "Banco"}},
		{name: "duplicate column", header: []string{"Data", "Data", "Valor", "Identificador"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewNubankParserIMPL(&tt.header)
			if err != nil {
				t.Fatalf("NewNubankParserIMPL() error = %v", err)
			}
			if parser.Supports() {
				t.Fatal("expected Supports() to return false")
			}
		})
	}
}

func TestNubankParserIMPL_GetColumns(t *testing.T) {
	testCases := []struct {
		name            string
		header          []string
		expectedColumns map[string]int
	}{
		{
			name:   "get columns from Nubank header",
			header: []string{"Data", "Descrição", "Valor", "Identificador"},
			expectedColumns: map[string]int{
				"date":        0,
				"description": 1,
				"amount":      2,
				"external_id": 3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := NewNubankParserIMPL(&tc.header)
			if err != nil {
				t.Fatalf("Failed to create NubankParserIMPL: %v", err)
			}

			if !parser.Supports() {
				t.Errorf("Expected Supports() to return true, but got false")
			}

			for key, expectedIndex := range tc.expectedColumns {
				if index, exists := parser.GetColumnIndex(key); !exists || index != expectedIndex {
					t.Errorf("Expected column '%s' to have index %d, but got %d", key, expectedIndex, index)
				}
			}

			if _, exists := parser.GetColumnIndex("missing"); exists {
				t.Error("expected missing column to be absent")
			}
		})
	}
}
