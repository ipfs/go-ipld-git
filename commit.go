package ipldgit

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/schema"
)

// DecodeCommit fills a NodeAssembler (from `Type.Commit__Repr.NewBuilder()`) from a stream of bytes
func DecodeCommit(na ipld.NodeAssembler, rd *bufio.Reader) error {
	size, err := rd.ReadString(0)
	if err != nil {
		return err
	}

	c := _Commit{
		Parents:  _ListParents{[]_Link{}},
		MergeTag: _ListTag{[]_Tag{}},
		Other:    _ListString{[]_String{}},
	}
	c.DataSize = _String{size}
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		err = decodeCommitLine(&c, line, rd)
		if err != nil {
			return err
		}
	}

	return na.AssignNode(&c)
}

func decodeCommitLine(c Commit, line []byte, rd *bufio.Reader) error {
	switch {
	case bytes.HasPrefix(line, []byte("tree ")):
		sha, err := hex.DecodeString(string(line[5:]))
		if err != nil {
			return err
		}

		c.GitTree = _LinkTree{cidlink.Link{Cid: shaToCid(sha)}}
	case bytes.HasPrefix(line, []byte("parent ")):
		psha, err := hex.DecodeString(string(line[7:]))
		if err != nil {
			return err
		}

		c.Parents.x = append(c.Parents.x, _Link{cidlink.Link{Cid: shaToCid(psha)}})
	case bytes.HasPrefix(line, []byte("author ")):
		a, err := parsePersonInfo(line)
		if err != nil {
			return err
		}

		c.Author = _PersonInfo__Maybe{m: schema.Maybe_Value, v: a}
	case bytes.HasPrefix(line, []byte("committer ")):
		com, err := parsePersonInfo(line)
		if err != nil {
			return err
		}

		c.Committer = _PersonInfo__Maybe{m: schema.Maybe_Value, v: com}
	case bytes.HasPrefix(line, []byte("encoding ")):
		c.Encoding = _String__Maybe{m: schema.Maybe_Value, v: &_String{string(line[9:])}}
	case bytes.HasPrefix(line, []byte("mergetag object ")):
		sha, err := hex.DecodeString(string(line)[16:])
		if err != nil {
			return err
		}

		mt, rest, err := readMergeTag(sha, rd)
		if err != nil {
			return err
		}

		c.MergeTag.x = append(c.MergeTag.x, *mt)

		if rest != nil {
			err = decodeCommitLine(c, rest, rd)
			if err != nil {
				return err
			}
		}
	case bytes.HasPrefix(line, []byte("gpgsig ")):
		sig, err := decodeGpgSig(rd)
		if err != nil {
			return err
		}
		c.Sig = _GpgSig__Maybe{m: schema.Maybe_Value, v: sig}
	case len(line) == 0:
		rest, err := ioutil.ReadAll(rd)
		if err != nil {
			return err
		}

		c.Message = _String{string(rest)}
	default:
		c.Other.x = append(c.Other.x, _String{string(line)})
	}
	return nil
}

func decodeGpgSig(rd *bufio.Reader) (GpgSig, error) {
	line, _, err := rd.ReadLine()
	if err != nil {
		return nil, err
	}

	out := _GpgSig{}

	if string(line) != " " {
		if strings.HasPrefix(string(line), " Version: ") || strings.HasPrefix(string(line), " Comment: ") {
			out.x += string(line) + "\n"
		} else {
			return nil, fmt.Errorf("expected first line of sig to be a single space or version")
		}
	} else {
		out.x += " \n"
	}

	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			return nil, err
		}

		if bytes.Equal(line, []byte(" -----END PGP SIGNATURE-----")) {
			break
		}

		out.x += string(line) + "\n"
	}

	return &out, nil
}

func encodeCommit(n ipld.Node, w io.Writer) error {
	c, ok := n.(Commit)
	if !ok {
		return fmt.Errorf("not a Commit: %T", n)
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "commit %s\x00", c.DataSize.x)
	fmt.Fprintf(buf, "tree %s\n", hex.EncodeToString(c.GitTree.sha()))
	for _, p := range c.Parents.x {
		fmt.Fprintf(buf, "parent %s\n", hex.EncodeToString(p.sha()))
	}
	fmt.Fprintf(buf, "author %s\n", c.Author.v.GitString())
	fmt.Fprintf(buf, "committer %s\n", c.Committer.v.GitString())
	if len(c.Encoding.v.x) > 0 {
		fmt.Fprintf(buf, "encoding %s\n", c.Encoding.v.x)
	}
	for _, mtag := range c.MergeTag.x {
		fmt.Fprintf(buf, "mergetag object %s\n", hex.EncodeToString(mtag.Object.sha()))
		fmt.Fprintf(buf, " type %s\n", mtag.TagType.x)
		fmt.Fprintf(buf, " tag %s\n", mtag.Tag.x)
		fmt.Fprintf(buf, " tagger %s\n \n", mtag.Tagger.GitString())
		fmt.Fprintf(buf, "%s", mtag.Text.x)
	}
	if c.Sig.m == schema.Maybe_Value {
		fmt.Fprintln(buf, "gpgsig -----BEGIN PGP SIGNATURE-----")
		fmt.Fprint(buf, c.Sig.v.x)
		fmt.Fprintln(buf, " -----END PGP SIGNATURE-----")
	}
	for _, line := range c.Other.x {
		fmt.Fprintln(buf, line.x)
	}
	fmt.Fprintf(buf, "\n%s", c.Message.x)

	_, err := bufio.NewWriter(w).Write(buf.Bytes())
	return err
}

func (p _PersonInfo) GitString() string {
	f := "%s <%s>"
	arg := []interface{}{p.Name.x, p.Email.x}
	if p.Date.x != "" {
		f = f + " %s"
		arg = append(arg, p.Date.x)
	}

	if p.Timezone.x != "" {
		f = f + " %s"
		arg = append(arg, p.Timezone.x)
	}
	return fmt.Sprintf(f, arg...)
}
