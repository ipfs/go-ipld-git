package ipldgit

import (
	"bufio"
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime"
)

// Decoder reads from a reader to fill a NodeAssembler
func Decoder(na ipld.NodeAssembler, r io.Reader) error {
	rd := bufio.NewReader(r)

	typ, err := rd.ReadString(' ')
	if err != nil {
		return err
	}
	typ = typ[:len(typ)-1]

	switch typ {
	case "tree":
		return DecodeTree(na, rd)
	case "commit":
		return DecodeCommit(na, rd)
	case "blob":
		return DecodeBlob(na, rd)
	case "tag":
		return DecodeTag(na, rd)
	default:
		return fmt.Errorf("unrecognized object type: %s", typ)
	}
}
