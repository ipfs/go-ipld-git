package ipldgit

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// DecodeTag fills a NodeAssembler (from `Type.Tag__Repr.NewBuilder()`) from a stream of bytes
func DecodeTag(na ipld.NodeAssembler, rd *bufio.Reader) error {
	_, err := rd.ReadString(0)
	if err != nil {
		return err
	}

	out := _Tag{}

	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch {
		case bytes.HasPrefix(line, []byte("object ")):
			sha, err := hex.DecodeString(string(line[7:]))
			if err != nil {
				return err
			}

			out.Object = _Link{cidlink.Link{Cid: shaToCid(sha)}}
		case bytes.HasPrefix(line, []byte("tag ")):
			out.Tag = _String{string(line[4:])}
		case bytes.HasPrefix(line, []byte("tagger ")):
			c, err := parsePersonInfo(line)
			if err != nil {
				return err
			}

			out.Tagger = *c
		case bytes.HasPrefix(line, []byte("type ")):
			out.TagType = _String{string(line[5:])}
		case len(line) == 0:
			rest, err := ioutil.ReadAll(rd)
			if err != nil {
				return err
			}

			out.Text = _String{string(rest)}
		default:
			fmt.Println("unhandled line: ", string(line))
		}
	}

	return na.AssignNode(&out)
}

// readMergeTag works for tags within commits like DecodeTag
func readMergeTag(hash []byte, rd *bufio.Reader) (Tag, []byte, error) {
	out := _Tag{}

	out.Object = _Link{cidlink.Link{Cid: shaToCid(hash)}}
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}

		switch {
		case bytes.HasPrefix(line, []byte(" type ")):
			out.TagType = _String{string(line[6:])}
		case bytes.HasPrefix(line, []byte(" tag ")):
			out.Tag = _String{string(line[5:])}
		case bytes.HasPrefix(line, []byte(" tagger ")):
			tagger, err := parsePersonInfo(line[1:])
			if err != nil {
				return nil, nil, err
			}
			out.Tagger = *tagger
		case string(line) == " ":
			for {
				line, _, err := rd.ReadLine()
				if err != nil {
					return nil, nil, err
				}

				if !bytes.HasPrefix(line, []byte(" ")) {
					return &out, line, nil
				}

				out.Text.x += string(line) + "\n"
			}
		}
	}
	return &out, nil, nil
}

func encodeTag(n ipld.Node, w io.Writer) error {
	obj, err := n.LookupByString("Object")
	if err != nil {
		return err
	}
	objLnk, err := obj.AsLink()
	if err != nil {
		return err
	}

	tt, err := n.LookupByString("TagType")
	if err != nil {
		return err
	}
	ttStr, err := tt.AsString()
	if err != nil {
		return err
	}

	tag, err := n.LookupByString("Tag")
	if err != nil {
		return err
	}
	tagStr, err := tag.AsString()
	if err != nil {
		return err
	}

	text, err := n.LookupByString("Text")
	if err != nil {
		return err
	}
	textStr, err := text.AsString()
	if err != nil {
		return err
	}

	tagger, taggerErr := n.LookupByString("Tagger")

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "object %s\n", hex.EncodeToString(sha(objLnk)))
	fmt.Fprintf(buf, "type %s\n", ttStr)
	fmt.Fprintf(buf, "tag %s\n", tagStr)
	if taggerErr == nil && !tagger.IsNull() {
		pi := Type.PersonInfo__Repr.NewBuilder()
		if err := pi.AssignNode(tagger); err != nil {
			return err
		}
		piN := pi.Build()
		parsed, ok := piN.(*_PersonInfo)
		if !ok {
			return fmt.Errorf("Could not parse tagger person info %v", tagger)
		}
		fmt.Fprintf(buf, "tagger %s\n", parsed.GitString())
	}
	if textStr != "" {
		fmt.Fprintf(buf, "\n%s", textStr)
	}

	if _, err := fmt.Fprintf(w, "tag %d\x00", buf.Len()); err != nil {
		return err
	}
	_, err = buf.WriteTo(w)
	return err
}
