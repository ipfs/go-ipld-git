package ipldgit

import (
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

func shaToCid(sha []byte) cid.Cid {
	h, _ := mh.Encode(sha, mh.SHA1)
	return cid.NewCidV1(cid.GitRaw, h)
}
