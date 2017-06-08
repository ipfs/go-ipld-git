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
	err := filepath.Walk(".git/objects", func(path string, info os.FileInfo, err error) error {
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

		fmt.Printf("%s\r", path)

		sha := thing.(GitObj).GitSha()
		if fmt.Sprintf("%x", sha) != parts[len(parts)-2]+parts[len(parts)-1] {
			fmt.Printf("\nsha: %x\n", sha)
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

func TestParsePersonInfo(t *testing.T) {
	pi, err := parsePersonInfo([]byte("prefix Someone <some@one.somewhere> 123456 +0123"))
	if err != nil {
		t.Fatal(err)
	}

	if pi.Date != "123456" {
		t.Fatalf("invalid date, got %s\n", pi.Date)
	}

	if pi.Timezone != "+0123" {
		t.Fatalf("invalid timezone, got %s\n", pi.Timezone)
	}

	if pi.Email != "some@one.somewhere" {
		t.Fatalf("invalid email, got %s\n", pi.Email)
	}

	if pi.Name != "Someone" {
		t.Fatalf("invalid name, got %s\n", pi.Name)
	}

	pi, err = parsePersonInfo([]byte("prefix So Me One <some@one.somewhere> 123456 +0123"))
	if err != nil {
		t.Fatal(err)
	}

	if pi.Name != "So Me One" {
		t.Fatalf("invalid name, got %s\n", pi.Name)
	}

	pi, err = parsePersonInfo([]byte("prefix Some One & Other One <some@one.somewhere, other@one.elsewhere> 987654 +4321"))
	if err != nil {
		t.Fatal(err)
	}

	if pi.Date != "987654" {
		t.Fatalf("invalid date, got %s\n", pi.Date)
	}

	if pi.Timezone != "+4321" {
		t.Fatalf("invalid timezone, got %s\n", pi.Timezone)
	}

	if pi.Email != "some@one.somewhere, other@one.elsewhere" {
		t.Fatalf("invalid email, got %s\n", pi.Email)
	}

	if pi.Name != "Some One & Other One" {
		t.Fatalf("invalid name, got %s\n", pi.Name)
	}

	pi, err = parsePersonInfo([]byte("prefix  <some@one.somewhere> 987654 +4321"))
	if err != nil {
		t.Fatal(err)
	}

	if pi.Name != "" {
		t.Fatalf("invalid name, got %s\n", pi.Name)
	}

	if pi.Email != "some@one.somewhere" {
		t.Fatalf("invalid email, got %s\n", pi.Email)
	}

	if pi.Date != "987654" {
		t.Fatalf("invalid date, got %s\n", pi.Date)
	}

	if pi.Timezone != "+4321" {
		t.Fatalf("invalid timezone, got %s\n", pi.Timezone)
	}

	pi, err = parsePersonInfo([]byte("prefix Someone  <some@one.somewhere> 987654 +4321"))
	if err != nil {
		t.Fatal(err)
	}

	if pi.Name != "Someone " {
		t.Fatalf("invalid name, got %s\n", pi.Name)
	}
}
