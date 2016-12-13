package ipldgit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type GitObj interface {
	GitSha() []byte
}

func TestObjectParse(t *testing.T) {
	err := filepath.Walk("samples/objects", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		parts := strings.Split(path, "/")
		dir := parts[len(parts)-2]
		if dir == "info" || dir == "pack" {
			return nil
		}

		fi, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fi.Close()

		thing, err := ParseCompressedObject(fi)
		if err != nil {
			fmt.Println("ERROR: ", path, err)
			return err
		}

		sha := thing.(GitObj).GitSha()
		if fmt.Sprintf("%x", sha) != parts[len(parts)-2]+parts[len(parts)-1] {
			fmt.Printf("sha: %x\n", sha)
			fmt.Printf("path: %s\n", path)
			fmt.Printf("mismatch on: %T\n", thing)
			fmt.Printf("%#v\n", thing)
			fmt.Println(string(thing.RawData()))
			t.Fatal("mismatch!")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
