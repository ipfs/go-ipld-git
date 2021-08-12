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
		FieldSymbolLowerOverrides: map[gengo.FieldTuple]string{
			{TypeName: "Tag", FieldName: "type"}: "typ",
		},
	}

	ts.Accumulate(schema.SpawnString("String"))
	ts.Accumulate(schema.SpawnLink("Link"))

	ts.Accumulate(schema.SpawnList("String_List", "String", false))

	ts.Accumulate(schema.SpawnStruct("PersonInfo", []schema.StructField{
		schema.SpawnStructField("date", "String", false, false),
		schema.SpawnStructField("timezone", "String", false, false),
		schema.SpawnStructField("email", "String", false, false),
		schema.SpawnStructField("name", "String", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))

	ts.Accumulate(schema.SpawnString("GpgSig"))

	ts.Accumulate(schema.SpawnStruct("Tag", []schema.StructField{
		schema.SpawnStructField("object", "Link", false, false),
		schema.SpawnStructField("type", "String", false, false),
		schema.SpawnStructField("tag", "String", false, false),
		schema.SpawnStructField("tagger", "PersonInfo", false, false),
		schema.SpawnStructField("message", "String", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))

	ts.Accumulate(schema.SpawnList("Tag_List", "Tag", false))

	ts.Accumulate(schema.SpawnStruct("Commit", []schema.StructField{
		schema.SpawnStructField("tree", "Tree_Link", false, false),
		schema.SpawnStructField("parents", "Commit_Link_List", false, false),
		schema.SpawnStructField("message", "String", false, false),
		schema.SpawnStructField("author", "PersonInfo", true, false),
		schema.SpawnStructField("committer", "PersonInfo", true, false),
		schema.SpawnStructField("encoding", "String", true, false),
		schema.SpawnStructField("signature", "GpgSig", true, false),
		schema.SpawnStructField("mergetag", "Tag_List", false, false),
		schema.SpawnStructField("other", "String_List", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))
	ts.Accumulate(schema.SpawnLinkReference("Commit_Link", "Commit"))
	ts.Accumulate(schema.SpawnList("Commit_Link_List", "Commit_Link", false))

	ts.Accumulate(schema.SpawnBytes("Blob"))

	ts.Accumulate(schema.SpawnMap("Tree", "String", "TreeEntry", false))
	ts.Accumulate(schema.SpawnStruct("TreeEntry", []schema.StructField{
		schema.SpawnStructField("mode", "String", false, false),
		schema.SpawnStructField("hash", "Link", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))
	ts.Accumulate(schema.SpawnLinkReference("Tree_Link", "Tree"))

	if errs := ts.ValidateGraph(); errs != nil {
		for _, err := range errs {
			fmt.Printf("- %s\n", err)
		}
		panic("not happening")
	}

	gengo.Generate(os.Args[1], "ipldgit", ts, adjCfg)
}
