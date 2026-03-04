package ipldgit

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// buildCommitRaw creates a raw git commit object with the given signature block.
// The sig parameter should be the complete gpgsig header (including the
// "gpgsig " prefix and trailing newline), or empty for an unsigned commit.
func buildCommitRaw(sig string) string {
	// Use a fixed tree hash (40 hex chars)
	tree := "4b825dc642cb6eb9a060e54bf899d69f7ef9c0b8"
	author := "Test User <test@example.com> 1234567890 +0000"
	committer := author
	message := "test commit"

	var b strings.Builder
	b.WriteString(fmt.Sprintf("tree %s\n", tree))
	b.WriteString(fmt.Sprintf("author %s\n", author))
	b.WriteString(fmt.Sprintf("committer %s\n", committer))
	b.WriteString(sig)
	b.WriteString(fmt.Sprintf("\n%s", message))

	content := b.String()
	return fmt.Sprintf("commit %d\x00%s", len(content), content)
}

// sigFixtures defines test signatures of various types. Each entry provides a
// complete gpgsig header block and a substring that must appear in the parsed
// GpgSig value.
var sigFixtures = []struct {
	name      string
	sig       string
	wantInSig string // substring expected in the parsed signature value
}{
	{
		name: "PGP",
		sig: "gpgsig -----BEGIN PGP SIGNATURE-----\n" +
			" \n" +
			" iQEzBAABCAAdFiEEtest+test+test+test+test+tE=\n" +
			" =ABCD\n" +
			" -----END PGP SIGNATURE-----\n",
		wantInSig: "-----BEGIN PGP SIGNATURE-----",
	},
	{
		name: "SSH",
		sig: "gpgsig -----BEGIN SSH SIGNATURE-----\n" +
			" U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgtestkey1234567890\n" +
			" abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOP==\n" +
			" -----END SSH SIGNATURE-----\n",
		wantInSig: "-----BEGIN SSH SIGNATURE-----",
	},
	{
		name: "X509",
		sig: "gpgsig -----BEGIN SIGNED MESSAGE-----\n" +
			" MIIBxjCCAWugAwIBAgIUTestCertData1234567890abcdef==\n" +
			" -----END SIGNED MESSAGE-----\n",
		wantInSig: "-----BEGIN SIGNED MESSAGE-----",
	},
}

func TestSignatureRoundTrip(t *testing.T) {
	for _, tt := range sigFixtures {
		t.Run(tt.name, func(t *testing.T) {
			raw := buildCommitRaw(tt.sig)

			nd, err := ParseObject(strings.NewReader(raw))
			if err != nil {
				t.Fatalf("ParseObject failed: %v", err)
			}

			var buf bytes.Buffer
			if err := Encode(nd, &buf); err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if buf.String() != raw {
				t.Errorf("round-trip mismatch.\n--- want ---\n%q\n--- got ---\n%q", raw, buf.String())
			}
		})
	}
}

func TestSignatureParsed(t *testing.T) {
	for _, tt := range sigFixtures {
		t.Run(tt.name, func(t *testing.T) {
			raw := buildCommitRaw(tt.sig)

			nd, err := ParseObject(strings.NewReader(raw))
			if err != nil {
				t.Fatalf("ParseObject failed: %v", err)
			}

			commit, ok := nd.(Commit)
			if !ok {
				t.Fatalf("expected Commit, got %T", nd)
			}

			sigNode, err := commit.LookupByString("signature")
			if err != nil {
				t.Fatalf("LookupByString(signature) failed: %v", err)
			}
			if sigNode.IsNull() {
				t.Fatal("signature is null")
			}
			sigStr, err := sigNode.AsString()
			if err != nil {
				t.Fatalf("AsString failed: %v", err)
			}
			if !strings.Contains(sigStr, tt.wantInSig) {
				t.Errorf("signature does not contain %q; got: %q", tt.wantInSig, sigStr)
			}
		})
	}
}

func TestPGPSignatureWithVersionRoundTrip(t *testing.T) {
	// PGP signatures sometimes include Version/Comment headers before the
	// blank separator line; this is PGP-specific but must still round-trip.
	sig := "gpgsig -----BEGIN PGP SIGNATURE-----\n" +
		" Version: GnuPG v1\n" +
		" Comment: some comment\n" +
		" \n" +
		" iQEzBAABCAAdFiEEtest+test+test+test+test+tE=\n" +
		" =ABCD\n" +
		" -----END PGP SIGNATURE-----\n"

	raw := buildCommitRaw(sig)

	nd, err := ParseObject(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseObject failed: %v", err)
	}

	var buf bytes.Buffer
	if err := Encode(nd, &buf); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if buf.String() != raw {
		t.Errorf("round-trip mismatch.\n--- want ---\n%q\n--- got ---\n%q", raw, buf.String())
	}
}

// TestNoSignatureRoundTrip ensures commits without signatures still work.
func TestNoSignatureRoundTrip(t *testing.T) {
	raw := buildCommitRaw("")

	nd, err := ParseObject(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseObject failed: %v", err)
	}

	var buf bytes.Buffer
	if err := Encode(nd, &buf); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if buf.String() != raw {
		t.Errorf("round-trip mismatch.\n--- want ---\n%q\n--- got ---\n%q", raw, buf.String())
	}

	commit := nd.(Commit)
	sigNode, err := commit.LookupByString("signature")
	if err != nil {
		t.Fatalf("LookupByString(signature) failed: %v", err)
	}
	if !sigNode.IsAbsent() {
		t.Error("expected absent signature for unsigned commit")
	}
}
