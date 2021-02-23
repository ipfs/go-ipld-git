package ipldgit

import (
	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

func (ip _Link__Prototype) fromString(l *_Link, s string) error {
	c, err := cid.Decode(s)
	if err != nil {
		return err
	}
	l.x = cidlink.Link{Cid: c}
	return nil
}

func (ip *_Link) String() string {
	return ip.String()
}
