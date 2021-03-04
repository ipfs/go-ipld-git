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

// ParseObject produces an ipld.Node from a stream / binary represnetation.
func ParseObject(r io.Reader) (ipld.Node, error) {
	rd := bufio.NewReader(r)

	typ, err := rd.ReadString(' ')
	if err != nil {
		return nil, err
	}
	typ = typ[:len(typ)-1]
	fmt.Printf("type is %s\n", typ)

	var na ipld.NodeBuilder
	var f func(ipld.NodeAssembler, *bufio.Reader) error
	switch typ {
	case "tree":
		na = Type.Tree.NewBuilder()
		f = DecodeTree
	case "commit":
		na = Type.Commit.NewBuilder()
		f = DecodeCommit
	case "blob":
		na = Type.Blob.NewBuilder()
		f = DecodeBlob
	case "tag":
		na = Type.Tag.NewBuilder()
		f = DecodeTag
	default:
		return nil, fmt.Errorf("unrecognized object type: %s", typ)
	}

	if err := f(na, rd); err != nil {
		return nil, err
	}
	return na.Build(), nil
}
