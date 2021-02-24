package ipldgit

//go:generate go run ./gen .
//go:generate go fmt ./

import (
	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

var (
	_ cidlink.MulticodecDecoder = Decoder
	_ cidlink.MulticodecEncoder = Encoder
)

func init() {
	cidlink.RegisterMulticodecDecoder(cid.GitRaw, Decoder)
	cidlink.RegisterMulticodecEncoder(cid.GitRaw, Encoder)
}
