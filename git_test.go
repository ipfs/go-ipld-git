package ipldgit

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
)

func TestObjectParse(t *testing.T) {
	lb := cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  1,
		Codec:    cid.GitRaw,
		MhType:   0x11,
		MhLength: 20,
	}}
	ls := cidlink.DefaultLinkSystem()

	i := 0
	err := filepath.Walk(".git/objects", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		parts := strings.Split(path, string(filepath.Separator))

		dir := parts[len(parts)-2]
		if dir == "info" || dir == "pack" {
			return nil
		}

		fi, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fi.Close()

		fmt.Printf("parsing for %s\n", path)
		thing, err := ParseCompressedObject(fi)
		if err != nil {
			fmt.Println("ERROR: ", path, err)
			return err
		}
		fmt.Printf("parsed.. now compute link.\n")

		if i%64 == 0 {
			fmt.Printf("%d %s\r", i, path)
		}

		lnk := ls.MustComputeLink(lb, thing)
		sha := lnk.(cidlink.Link).Cid.Hash()
		mh, err := multihash.Decode(sha)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", mh.Digest) != parts[len(parts)-2]+parts[len(parts)-1] {
			fmt.Printf("\n")
			fmt.Printf("\nsha: %x\n", sha)
			fmt.Printf("path: %s\n", path)
			fmt.Printf("mismatch on: %T\n", thing)
			fmt.Printf("%#v\n", thing)
			fmt.Println("vvvvvv")
			fmt.Println(thing.AsBytes())
			fmt.Println("^^^^^^")
			t.Fatal("mismatch!")
		} else {
			fmt.Printf("\nsha: %x\n", mh.Digest)
		}

		err = testNode(t, thing)
		if err != nil {
			t.Fatalf("error: %s, %s", path, err)
		}
		i++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestArchiveObjectParse(t *testing.T) {
	lb := cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  1,
		Codec:    cid.GitRaw,
		MhType:   0x11,
		MhLength: 20,
	}}
	ls := cidlink.DefaultLinkSystem()

	archive, err := os.Open("testdata.tar.gz")
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	defer archive.Close()

	gz, err := gzip.NewReader(archive)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	tarReader := tar.NewReader(gz)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println("ERROR: ", err)
			return
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if !strings.HasPrefix(name, ".git/objects/") {
				continue
			}

			parts := strings.Split(name, "/")
			dir := parts[2]
			if dir == "info" || dir == "pack" {
				continue
			}

			thing, err := ParseCompressedObject(tarReader)
			if err != nil {
				fmt.Println("ERROR: ", name, err)
				return
			}

			fmt.Printf("%s\r", name)

			lnk := ls.MustComputeLink(lb, thing)
			sha := lnk.(cidlink.Link).Cid.Hash()
			mh, err := multihash.Decode(sha)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%x", mh.Digest) != parts[len(parts)-2]+parts[len(parts)-1] {
				fmt.Printf("\nsha: %x\n", sha)
				fmt.Printf("path: %s\n", name)
				fmt.Printf("mismatch on: %T\n", thing)
				fmt.Printf("%#v\n", thing)
				fmt.Println("vvvvvv")
				buf := bytes.NewBuffer([]byte{})
				Encode(thing, buf)
				fmt.Println(buf.String())
				fmt.Println("^^^^^^")
				t.Fatal("mismatch!")
			}

			err = testNode(t, thing)
			if err != nil {
				t.Fatalf("error: %s, %s", name, err)
			}
		default:

		}
	}

}

func testNode(t *testing.T, nd ipld.Node) error {
	switch nd.Prototype() {
	case Type.Blob:
		blob, ok := nd.(Blob)
		if !ok {
			t.Fatalf("Blob is not a blob")
		}

		b, err := blob.AsBytes()
		assert(t, err == nil)
		assert(t, len(b) != 0)

	case Type.Commit:
		commit, ok := nd.(Commit)
		if !ok {
			t.Fatalf("Commit is not a commit")
		}

		assert(t, !commit.tree.IsNull())
	case Type.Tag:
		tag, ok := nd.(Tag)
		if !ok {
			t.Fatalf("Tag is not a tag")
		}

		tt, err := tag.typ.AsString()
		assert(t, err == nil)

		assert(t, tt == "commit" || tt == "tree" || tt == "blob" || tt == "tag")
		assert(t, !tag.object.IsNull())

	case Type.Tree:
		tree, ok := nd.(Tree)
		if !ok {
			t.Fatalf("Tree is not a tree")
		}

		assert(t, len(tree.m) > 0)

	default:
		return fmt.Errorf("unexpected or unknown NodePrototype %v", nd.Prototype())
	}

	return nil
}

func TestParsePersonInfo(t *testing.T) {
	p1 := []byte("prefix Someone <some@one.somewhere> 123456 +0123")
	pi, err := parsePersonInfo(p1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(pi.GitString()), p1[7:]) {
		t.Fatal("not equal:", string(p1), "vs: ", pi.GitString())
	}

	if d, err := pi.LookupByString("date"); err != nil {
		t.Fatalf("invalid date, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "123456" {
		t.Fatalf("invalid date, got %s\n", ds)
	}

	if d, err := pi.LookupByString("timezone"); err != nil {
		t.Fatalf("invalid timezone, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "+0123" {
		t.Fatalf("invalid timezone, got %s\n", ds)
	}

	if d, err := pi.LookupByString("email"); err != nil {
		t.Fatalf("invalid email, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "some@one.somewhere" {
		t.Fatalf("invalid email, got %s\n", ds)
	}

	if d, err := pi.LookupByString("name"); err != nil {
		t.Fatalf("invalid name, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "Someone" {
		t.Fatalf("invalid name, got %s\n", ds)
	}

	p2 := []byte("prefix So Me One <some@one.somewhere> 123456 +0123")
	pi, err = parsePersonInfo(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(pi.GitString()), p2[7:]) {
		t.Fatal("not equal", p2, pi.GitString())
	}

	if d, err := pi.LookupByString("name"); err != nil {
		t.Fatalf("invalid name, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "So Me One" {
		t.Fatalf("invalid name, got %s\n", ds)
	}

	p3 := []byte("prefix Some One & Other One <some@one.somewhere, other@one.elsewhere> 987654 +4321")
	pi, err = parsePersonInfo(p3)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(pi.GitString()), p3[7:]) {
		t.Fatal("not equal", p3, pi.GitString())
	}
	if d, err := pi.LookupByString("date"); err != nil {
		t.Fatalf("invalid date, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "987654" {
		t.Fatalf("invalid date, got %s\n", ds)
	}

	if d, err := pi.LookupByString("timezone"); err != nil {
		t.Fatalf("invalid tz, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "+4321" {
		t.Fatalf("invalid tz, got %s\n", ds)
	}

	if d, err := pi.LookupByString("email"); err != nil {
		t.Fatalf("invalid email, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "some@one.somewhere, other@one.elsewhere" {
		t.Fatalf("invalid email, got %s\n", ds)
	}

	if d, err := pi.LookupByString("name"); err != nil {
		t.Fatalf("invalid name, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "Some One & Other One" {
		t.Fatalf("invalid name, got %s\n", ds)
	}

	p4 := []byte("prefix  <some@one.somewhere> 987654 +4321")
	pi, err = parsePersonInfo(p4)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(pi.GitString()), p4[7:]) {
		t.Fatal("not equal", p4, pi.GitString())
	}

	if d, err := pi.LookupByString("name"); err != nil {
		t.Fatalf("invalid name, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "" {
		t.Fatalf("invalid name, got %s\n", ds)
	}

	if d, err := pi.LookupByString("email"); err != nil {
		t.Fatalf("invalid email, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "some@one.somewhere" {
		t.Fatalf("invalid email, got %s\n", ds)
	}

	if d, err := pi.LookupByString("date"); err != nil {
		t.Fatalf("invalid date, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "987654" {
		t.Fatalf("invalid date, got %s\n", ds)
	}

	if d, err := pi.LookupByString("timezone"); err != nil {
		t.Fatalf("invalid tz, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "+4321" {
		t.Fatalf("invalid tz, got %s\n", ds)
	}

	p5 := []byte("prefix Someone  <some@one.somewhere> 987654 +4321")
	pi, err = parsePersonInfo(p5)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(pi.GitString()), p5[7:]) {
		t.Fatal("not equal", p5, pi.GitString())
	}

	if d, err := pi.LookupByString("name"); err != nil {
		t.Fatalf("invalid name, got %s\n", err)
	} else if ds, _ := d.AsString(); ds != "Someone " {
		t.Fatalf("invalid name, got %s\n", ds)
	}

	p6 := []byte("prefix Someone <some.one@some.where>")
	pi, err = parsePersonInfo(p6)
	if err != nil {
		t.Fatal(err)
	}

	assert(t, pi.GitString() == "Someone <some.one@some.where>")

	buf := new(bytes.Buffer)

	pi, err = parsePersonInfo([]byte("prefix Łukasz Magiera <magik6k@users.noreply.github.com> 1546187652 +0100"))
	assert(t, err == nil)
	buf.Reset()
	err = dagjson.Encode(pi, buf)
	assert(t, err == nil)
	assert(t, buf.String() == `{"date":"1546187652","email":"magik6k@users.noreply.github.com","name":"Łukasz Magiera","timezone":"+0100"}`)

	pi, err = parsePersonInfo([]byte("prefix Sameer <sameer@users.noreply.github.com> 1545162499 -0500"))
	assert(t, err == nil)
	buf.Reset()
	err = dagjson.Encode(pi, buf)
	assert(t, err == nil)
	assert(t, buf.String() == `{"date":"1545162499","email":"sameer@users.noreply.github.com","name":"Sameer","timezone":"-0500"}`)

	pi, err = parsePersonInfo([]byte("prefix Łukasz Magiera <magik6k@users.noreply.github.com> 1546187652 +0122"))
	assert(t, err == nil)
	buf.Reset()
	err = dagjson.Encode(pi, buf)
	assert(t, err == nil)
	assert(t, buf.String() == `{"date":"1546187652","email":"magik6k@users.noreply.github.com","name":"Łukasz Magiera","timezone":"+0122"}`)

	pi, err = parsePersonInfo([]byte("prefix Sameer <sameer@users.noreply.github.com> 1545162499 -0545"))
	assert(t, err == nil)
	buf.Reset()
	err = dagjson.Encode(pi, buf)
	assert(t, err == nil)
	assert(t, buf.String() == `{"date":"1545162499","email":"sameer@users.noreply.github.com","name":"Sameer","timezone":"-0545"}`)
}

func assert(t *testing.T, ok bool) {
	if !ok {
		fmt.Printf("\n")
		panic("Assertion failed")
	}
}

func BenchmarkRawData(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := filepath.Walk(".git/objects", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			parts := strings.Split(path, string(filepath.Separator))
			if dir := parts[len(parts)-2]; dir == "info" || dir == "pack" {
				return nil
			}

			fi, err := os.Open(path)
			if err != nil {
				return err
			}

			thing, err := ParseCompressedObject(fi)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer([]byte{})
			return Encode(thing, buf)
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCid(b *testing.B) {
	lb := cidlink.LinkPrototype{Prefix: cid.Prefix{
		Version:  1,
		Codec:    cid.GitRaw,
		MhType:   0x11,
		MhLength: 20,
	}}
	ls := cidlink.DefaultLinkSystem()

	for i := 0; i < b.N; i++ {
		err := filepath.Walk(".git/objects", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			parts := strings.Split(path, string(filepath.Separator))
			if dir := parts[len(parts)-2]; dir == "info" || dir == "pack" {
				return nil
			}

			fi, err := os.Open(path)
			if err != nil {
				return err
			}

			thing, err := ParseCompressedObject(fi)
			if err != nil {
				return err
			}

			_, err = ls.ComputeLink(lb, thing)
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
