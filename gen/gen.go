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
		schema.SpawnStructField("Name", "String", false, false),
		schema.SpawnStructField("Email", "String", false, false),
		schema.SpawnStructField("Date", "String", false, false),
		schema.SpawnStructField("Timezone", "String", false, false),
	}, schema.SpawnStructRepresentationStringjoin(" ")))
	ts.Accumulate(schema.SpawnString("GpgSig"))
	ts.Accumulate(schema.SpawnStruct("Tag", []schema.StructField{
		schema.SpawnStructField("Object", "Link", false, false),
		schema.SpawnStructField("TagType", "String", false, false),
		schema.SpawnStructField("Tag", "String", false, false),
		schema.SpawnStructField("Tagger", "PersonInfo", false, false),
		schema.SpawnStructField("Text", "String", false, false),
		schema.SpawnStructField("DataSize", "String", true, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{"Type": "TagType"})))
	ts.Accumulate(schema.SpawnList("ListTag", "Tag", false))
	ts.Accumulate(schema.SpawnList("ListParents", "Link", false)) //Todo: type 'Parents' links
	ts.Accumulate(schema.SpawnStruct("Commit", []schema.StructField{
		schema.SpawnStructField("DataSize", "String", false, false),
		schema.SpawnStructField("GitTree", "LinkTree", false, false),
		schema.SpawnStructField("Parents", "ListParents", false, false),
		schema.SpawnStructField("Message", "String", false, false),
		schema.SpawnStructField("Author", "PersonInfo", true, false),
		schema.SpawnStructField("Committer", "PersonInfo", true, false),
		schema.SpawnStructField("Encoding", "String", true, false),
		schema.SpawnStructField("Sig", "GpgSig", true, false),
		schema.SpawnStructField("MergeTag", "ListTag", false, false),
		schema.SpawnStructField("Other", "ListString", false, false),
	}, schema.SpawnStructRepresentationMap(map[string]string{})))
	ts.Accumulate(schema.SpawnBytes("Blob"))

	ts.Accumulate(schema.SpawnList("Tree", "TreeEntry", false))
	ts.Accumulate(schema.SpawnLinkReference("LinkTree", "Tree"))
	ts.Accumulate(schema.SpawnStruct("TreeEntry", []schema.StructField{
		schema.SpawnStructField("Mode", "String", false, false),
		schema.SpawnStructField("Name", "String", false, false),
		schema.SpawnStructField("Hash", "Link", false, false),
	}, schema.SpawnStructRepresentationStringjoin(" ")))

	if errs := ts.ValidateGraph(); errs != nil {
		for _, err := range errs {
			fmt.Printf("- %s\n", err)
		}
		panic("not happening")
	}

	gengo.Generate(os.Args[1], "ipldgit", ts, adjCfg)
}
