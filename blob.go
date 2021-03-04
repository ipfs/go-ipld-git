package ipldgit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/ipld/go-ipld-prime"
)

// DecodeBlob fills a NodeAssembler (from `Type.Blob__Repr.NewBuilder()`) from a stream of bytes
func DecodeBlob(na ipld.NodeAssembler, rd *bufio.Reader) error {
	size, err := rd.ReadString(0)
	if err != nil {
		return err
	}
	fmt.Printf("Size of blob was %s\n", size)

	sizen, err := strconv.Atoi(size[:len(size)-1])
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "blob %d\x00", sizen)

	n, err := io.Copy(buf, rd)
	if err != nil {
		return err
	}

	if n != int64(sizen) {
		return fmt.Errorf("blob size was not accurate")
	}

	return na.AssignBytes(buf.Bytes())
}

func encodeBlob(n ipld.Node, w io.Writer) error {
	bytes, err := n.AsBytes()
	if err != nil {
		return err
	}
	_, err = bufio.NewWriter(w).Write(bytes)
	return err
}
