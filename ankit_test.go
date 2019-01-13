package ankit

import (
	"bytes"
	"testing"
)

type MockNote []string

func (m MockNote) Fields() []string {
	return []string(m)
}

func TestOneNoteReader(t *testing.T) {
	var tests = []struct {
		fields []string
		csv    string
	}{
		{
			[]string{"field1", "field2", "field3"},
			"field1,field2,field3\n",
		},
		{
			[]string{"field1", "field2\n\"content\"", "field3"},
			"field1,\"field2\n\"\"content\"\"\",field3\n",
		},
	}

	var buf bytes.Buffer
	for _, tt := range tests {
		buf.Reset()
		r := MockNote(tt.fields)
		Copy(&buf, OneNoteReader(r))

		if buf.String() != tt.csv {
			t.Errorf("got %q, want %q", buf.String(), tt.csv)
		}
	}
}
