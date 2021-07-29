package main

import (
	"fmt"
	"os"

	"github.com/ipld/go-ipld-prime/schema"
	gengo "github.com/ipld/go-ipld-prime/schema/gen/go"
)

func main() {
	ts := schema.TypeSystem{}
	ts.Init()
	adjCfg := &gengo.AdjunctCfg{
		CfgUnionMemlayout: map[schema.TypeName]string{},
	}
	ts.Accumulate(schema.SpawnString("String"))
	ts.Accumulate(schema.SpawnList("ListString", "String", false))
	ts.Accumulate(schema.SpawnLink("Link"))
	ts.Accumulate(schema.SpawnStruct("PersonInfo", []schema.StructField{
		schema.SpawnStructField("name", "String", false, false),
		schema.SpawnStructField("email", "String", false, false),
		schema.SpawnStructField("date", "String", false, false),
		schema.SpawnStructField("timezone", "String", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))
	ts.Accumulate(schema.SpawnString("GpgSig"))
	ts.Accumulate(schema.SpawnStruct("Tag", []schema.StructField{
		schema.SpawnStructField("object", "Link", false, false),
		schema.SpawnStructField("tagType", "String", false, false),
		schema.SpawnStructField("tag", "String", false, false),
		schema.SpawnStructField("tagger", "PersonInfo", false, false),
		schema.SpawnStructField("text", "String", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))
	ts.Accumulate(schema.SpawnList("ListTag", "Tag", false))
	ts.Accumulate(schema.SpawnLinkReference("LinkCommit", "Commit"))
	ts.Accumulate(schema.SpawnList("ListParents", "LinkCommit", false))
	ts.Accumulate(schema.SpawnStruct("Commit", []schema.StructField{
		schema.SpawnStructField("tree", "LinkTree", false, false),
		schema.SpawnStructField("parents", "ListParents", false, false),
		schema.SpawnStructField("message", "String", false, false),
		schema.SpawnStructField("author", "PersonInfo", true, false),
		schema.SpawnStructField("committer", "PersonInfo", true, false),
		schema.SpawnStructField("encoding", "String", true, false),
		schema.SpawnStructField("signature", "GpgSig", true, false),
		schema.SpawnStructField("mergeTag", "ListTag", false, false),
		schema.SpawnStructField("other", "ListString", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))
	ts.Accumulate(schema.SpawnBytes("Blob"))
	ts.Accumulate(schema.SpawnMap("Tree", "String", "TreeEntry", false))
	ts.Accumulate(schema.SpawnLinkReference("LinkTree", "Tree"))
	ts.Accumulate(schema.SpawnStruct("TreeEntry", []schema.StructField{
		schema.SpawnStructField("mode", "String", false, false),
		schema.SpawnStructField("hash", "Link", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))

	if errs := ts.ValidateGraph(); errs != nil {
		for _, err := range errs {
			fmt.Printf("- %s\n", err)
		}
		panic("not happening")
	}

	gengo.Generate(os.Args[1], "ipldgit", ts, adjCfg)
}
