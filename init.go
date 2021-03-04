package ipldgit

//go:generate go run ./gen .
//go:generate go fmt ./

import (
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	mc "github.com/ipld/go-ipld-prime/multicodec"
)

var (
	_ ipld.Decoder = Decoder
	_ ipld.Encoder = Encoder
)

func init() {
	mc.EncoderRegistry[cid.GitRaw] = Encoder
	mc.DecoderRegistry[cid.GitRaw] = Decoder
}
