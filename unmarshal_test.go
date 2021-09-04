package ipldgit

import (
	"fmt"
	"strings"
	"testing"

	basicnode "github.com/ipld/go-ipld-prime/node/basic"
)

func TestUnmarshalError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Empty", "", "unexpected EOF"},
		{"Whitespace", "  ", "unrecognized object type"},
		{"NoSpace", "foo", "unexpected EOF"},
		{"BadType", "foo ", "unrecognized object type"},
	}
	for _, test := range tests {
		t.Run("Decode/"+test.name, func(t *testing.T) {
			nb := basicnode.Prototype.Any.NewBuilder()
			err := Decode(nb, strings.NewReader(test.input))
			got := fmt.Sprint(err)
			if !strings.Contains(got, test.want) {
				t.Fatalf("Decode(%q) got %q, want %q", test.input, got, test.want)
			}
		})
		t.Run("ParseObject/"+test.name, func(t *testing.T) {
			_, err := ParseObject(strings.NewReader(test.input))
			got := fmt.Sprint(err)
			if !strings.Contains(got, test.want) {
				t.Fatalf("ParseObject(%q) got %q, want %q", test.input, got, test.want)
			}
		})
	}
}
