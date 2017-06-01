package ipldgit

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	cid "github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-node"
	mh "github.com/multiformats/go-multihash"
)

func ParseObjectFromBuffer(b []byte) (node.Node, error) {
	return ParseObject(bytes.NewReader(b))
}

func ParseCompressedObject(r io.Reader) (node.Node, error) {
	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return ParseObject(rc)
}

func ParseObject(r io.Reader) (node.Node, error) {
	rd := bufio.NewReader(r)

	typ, err := rd.ReadString(' ')
	if err != nil {
		return nil, err
	}
	typ = typ[:len(typ)-1]

	switch typ {
	case "tree":
		return ReadTree(rd)
	case "commit":
		return ReadCommit(rd)
	case "blob":
		return ReadBlob(rd)
	case "tag":
		return ReadTag(rd)
	default:
		return nil, fmt.Errorf("unrecognized type: %s", typ)
	}
}

func ReadBlob(rd *bufio.Reader) (Blob, error) {
	size, err := rd.ReadString(0)
	if err != nil {
		return nil, err
	}

	sizen, err := strconv.Atoi(size[:len(size)-1])
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "blob %d\x00", sizen)

	n, err := io.Copy(buf, rd)
	if err != nil {
		return nil, err
	}

	if n != int64(sizen) {
		return nil, fmt.Errorf("blob size was not accurate")
	}

	return Blob(buf.Bytes()), nil
}

func ReadCommit(rd *bufio.Reader) (*Commit, error) {
	size, err := rd.ReadString(0)
	if err != nil {
		return nil, err
	}

	out := &Commit{
		DataSize: size[:len(size)-1],
	}

	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch {
		case bytes.HasPrefix(line, []byte("tree ")):
			sha, err := hex.DecodeString(string(line[5:]))
			if err != nil {
				return nil, err
			}

			out.GitTree = shaToCid(sha)
		case bytes.HasPrefix(line, []byte("parent ")):
			psha, err := hex.DecodeString(string(line[7:]))
			if err != nil {
				return nil, err
			}

			out.Parents = append(out.Parents, shaToCid(psha))
		case bytes.HasPrefix(line, []byte("author ")):
			a, err := parsePersonInfo(line)
			if err != nil {
				return nil, err
			}

			out.Author = a
		case bytes.HasPrefix(line, []byte("committer ")):
			c, err := parsePersonInfo(line)
			if err != nil {
				return nil, err
			}

			out.Committer = c
		case bytes.HasPrefix(line, []byte("gpgsig ")):
			sig, err := ReadGpgSig(rd)
			if err != nil {
				return nil, err
			}
			out.Sig = sig

		case len(line) == 0:
			rest, err := ioutil.ReadAll(rd)
			if err != nil {
				return nil, err
			}

			out.Message = string(rest)
		default:
			fmt.Println("unhandled line: ", string(line))
		}
	}

	out.cid = hashObject(out.RawData())

	return out, nil
}

func ReadTag(rd *bufio.Reader) (*Tag, error) {
	size, err := rd.ReadString(0)
	if err != nil {
		return nil, err
	}

	out := &Tag{
		dataSize: size[:len(size)-1],
	}

	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch {
		case bytes.HasPrefix(line, []byte("object ")):
			sha, err := hex.DecodeString(string(line[7:]))
			if err != nil {
				return nil, err
			}

			out.Object = shaToCid(sha)
		case bytes.HasPrefix(line, []byte("tag ")):
			out.Tag = string(line[4:])
		case bytes.HasPrefix(line, []byte("tagger ")):
			c, err := parsePersonInfo(line)
			if err != nil {
				return nil, err
			}

			out.Tagger = c
		case bytes.HasPrefix(line, []byte("type ")):
			out.Type = string(line[5:])
		case len(line) == 0:
			rest, err := ioutil.ReadAll(rd)
			if err != nil {
				return nil, err
			}

			out.Message = string(rest)
		default:
			fmt.Println("unhandled line: ", string(line))
		}
	}

	out.cid = hashObject(out.RawData())

	return out, nil
}

func hashObject(data []byte) *cid.Cid {
	c, err := cid.Prefix{
		MhType:   mh.SHA1,
		MhLength: -1,
		Codec:    0x78, //TODO: change to cid.Git
		Version:  1,
	}.Sum(data)
	if err != nil {
		panic(err)
	}
	return c
}

func ReadGpgSig(rd *bufio.Reader) (*GpgSig, error) {
	line, _, err := rd.ReadLine()
	if err != nil {
		return nil, err
	}

	if string(line) != " " {
		return nil, fmt.Errorf("expected first line of sig to be a single space")
	}

	out := new(GpgSig)
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			return nil, err
		}

		if bytes.Equal(line, []byte(" -----END PGP SIGNATURE-----")) {
			break
		}

		out.Text += string(line) + "\n"
	}

	return out, nil
}

func parsePersonInfo(line []byte) (PersonInfo, error) {
	parts := bytes.Split(line, []byte{' '})
	if len(parts) < 5 {
		fmt.Println(string(line))
		return PersonInfo{}, fmt.Errorf("incorrectly formatted person info line")
	}

	var pi PersonInfo
	email_bytes := parts[len(parts)-3][1:]
	pi.Email = string(email_bytes[:len(email_bytes)-1])
	pi.Date = string(parts[len(parts)-2])
	pi.Timezone = string(parts[len(parts)-1])

	lb := len(parts[0]) + 1
	hb := len(line) - (len(pi.Email) + len(pi.Date) + len(pi.Timezone) + 5)
	pi.Name = string(line[lb:hb])
	return pi, nil
}

func ReadTree(rd *bufio.Reader) (*Tree, error) {
	lstr, err := rd.ReadString(0)
	if err != nil {
		return nil, err
	}
	lstr = lstr[:len(lstr)-1]

	n, err := strconv.Atoi(lstr)
	if err != nil {
		return nil, err
	}

	t := &Tree{
		entries: make(map[string]*TreeEntry),
		size:    n,
	}
	var order []string
	for {
		e, err := ReadEntry(rd)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		order = append(order, e.name)
		t.entries[e.name] = e
	}
	t.order = order
	t.cid = hashObject(t.RawData())

	return t, nil
}

func cidToSha(c *cid.Cid) []byte {
	h := c.Hash()
	return h[len(h)-20:]
}

func shaToCid(sha []byte) *cid.Cid {
	h, _ := mh.Encode(sha, mh.SHA1)
	return cid.NewCidV1(0x78, h) //TODO: change to cid.Git
}

func ReadEntry(r *bufio.Reader) (*TreeEntry, error) {
	data, err := r.ReadString(' ')
	if err != nil {
		return nil, err
	}
	data = data[:len(data)-1]

	name, err := r.ReadString(0)
	if err != nil {
		return nil, err
	}
	name = name[:len(name)-1]

	sha := make([]byte, 20)
	_, err = io.ReadFull(r, sha)
	if err != nil {
		return nil, err
	}

	return &TreeEntry{
		name: name,
		Mode: data,
		Hash: shaToCid(sha),
	}, nil
}
