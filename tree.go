package ipldgit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// DecodeTree fills a NodeAssembler (from `Type.Tree__Repr.NewBuilder()`) from a stream of bytes
func DecodeTree(na ipld.NodeAssembler, rd *bufio.Reader) error {
	lstr, err := rd.ReadString(0)
	if err != nil {
		return err
	}
	lstr = lstr[:len(lstr)-1]

	n, err := strconv.Atoi(lstr)
	if err != nil {
		return err
	}

	t := Type.Tree__Repr.NewBuilder()
	la, err := t.BeginList(int64(n))
	if err != nil {
		return err
	}
	for {
		err := DecodeTreeEntry(la.AssembleValue(), rd)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return na.AssignNode(t.Build())
}

// DecodeTreeEntry fills a NodeAssembler (from `Type.TreeEntry__Repr.NewBuilder()`) from a stream of bytes
func DecodeTreeEntry(na ipld.NodeAssembler, rd *bufio.Reader) error {
	data, err := rd.ReadString(' ')
	if err != nil {
		return err
	}
	data = data[:len(data)-1]

	name, err := rd.ReadString(0)
	if err != nil {
		return err
	}
	name = name[:len(name)-1]

	sha := make([]byte, 20)
	_, err = io.ReadFull(rd, sha)
	if err != nil {
		return err
	}

	te := _TreeEntry{
		Mode: _String{data},
		Name: _String{name},
		Hash: _Link{cidlink.Link{Cid: shaToCid(sha)}},
	}
	return na.AssignNode(&te)
}

func encodeTree(n ipld.Node, w io.Writer) error {
	buf := new(bytes.Buffer)

	cnt := 0
	li := n.ListIterator()
	for !li.Done() {
		_, te, err := li.Next()
		if err != nil {
			return err
		}
		cnt++
		if err := encodeTreeEntry(te, buf); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "tree %d\x00", cnt); err != nil {
		return err
	}

	_, err := bufio.NewWriter(w).Write(buf.Bytes())
	return err

}

func encodeTreeEntry(n ipld.Node, w io.Writer) error {
	m, err := n.LookupByString("Mode")
	if err != nil {
		return err
	}
	ms, err := m.AsString()
	if err != nil {
		return err
	}
	na, err := n.LookupByString("Name")
	if err != nil {
		return err
	}
	ns, err := na.AsString()
	if err != nil {
		return err
	}
	ha, err := n.LookupByString("Hash")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s %s\x00", ms, ns)
	if err != nil {
		return err
	}

	hal, err := ha.AsLink()
	if err != nil {
		return err
	}
	_, err = w.Write(cidToSha(hal.(cidlink.Link).Cid))
	if err != nil {
		return err
	}

	return nil
}
