package sync

import (
	"strings"
	"testing"
)

func TestParseCSV_Basic(t *testing.T) {
	input := `long-hair,general,appearance,"/lh,longhair"
touhou,copyright,anime_manga,"/to,toho"
highres,meta,quality_resolution,"high-res,high-resolution,hires"
simple,tag,sub,`

	r := strings.NewReader(input)
	tags, err := ParseCSV(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) != 4 {
		t.Fatalf("got %d tags, want 4", len(tags))
	}

	checks := []struct {
		name, cat, sub, aliases string
	}{
		{"long-hair", "general", "appearance", "/lh,longhair"},
		{"touhou", "copyright", "anime_manga", "/to,toho"},
		{"highres", "meta", "quality_resolution", "high-res,high-resolution,hires"},
		{"simple", "tag", "sub", ""},
	}

	for i, c := range checks {
		if tags[i].TagName != c.name {
			t.Errorf("tag[%d].TagName = %q, want %q", i, tags[i].TagName, c.name)
		}
		if tags[i].CategoryName != c.cat {
			t.Errorf("tag[%d].CategoryName = %q, want %q", i, tags[i].CategoryName, c.cat)
		}
		if tags[i].SubcategoryName != c.sub {
			t.Errorf("tag[%d].SubcategoryName = %q, want %q", i, tags[i].SubcategoryName, c.sub)
		}
		if tags[i].Aliases != c.aliases {
			t.Errorf("tag[%d].Aliases = %q, want %q", i, tags[i].Aliases, c.aliases)
		}
	}
}

func TestParseCSV_EmptyInput(t *testing.T) {
	r := strings.NewReader("")
	tags, err := ParseCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 {
		t.Fatalf("got %d tags, want 0", len(tags))
	}
}

func TestParseCSV_EmptyTagName(t *testing.T) {
	input := `,general,sub,aliases
valid,general,sub,`

	r := strings.NewReader(input)
	tags, err := ParseCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(tags))
	}
	if tags[0].TagName != "valid" {
		t.Errorf("TagName = %q, want valid", tags[0].TagName)
	}
}

func TestParseCSV_MissingColumns(t *testing.T) {
	input := `only-one-field`

	r := strings.NewReader(input)
	_, err := ParseCSV(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCSV_QuotedFieldWithComma(t *testing.T) {
	input := `tag,cat,sub,"alias1,alias2,alias3"`
	r := strings.NewReader(input)
	tags, err := ParseCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(tags))
	}
	if tags[0].Aliases != "alias1,alias2,alias3" {
		t.Errorf("Aliases = %q, want alias1,alias2,alias3", tags[0].Aliases)
	}
}

func TestParseCSV_UTF8(t *testing.T) {
	input := `café,general,sub,`
	r := strings.NewReader(input)
	tags, err := ParseCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 {
		t.Fatalf("got %d tags", len(tags))
	}
	if tags[0].TagName != "café" {
		t.Errorf("TagName = %q, want café", tags[0].TagName)
	}
}

func TestParseCSV_TrimWhitespace(t *testing.T) {
	input := `  spaced-tag  ,  general  ,  sub  ,  alias1  `
	r := strings.NewReader(input)
	tags, err := ParseCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if tags[0].TagName != "spaced-tag" {
		t.Errorf("TagName = %q", tags[0].TagName)
	}
	if tags[0].CategoryName != "general" {
		t.Errorf("CategoryName = %q", tags[0].CategoryName)
	}
	if tags[0].SubcategoryName != "sub" {
		t.Errorf("SubcategoryName = %q", tags[0].SubcategoryName)
	}
	if tags[0].Aliases != "alias1" {
		t.Errorf("Aliases = %q", tags[0].Aliases)
	}
}

func TestParseTXT_Basic(t *testing.T) {
	input := "tag1\ntag2\ntag3\n"
	r := strings.NewReader(input)
	tags, err := ParseTXT(r, "armor", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Fatalf("got %d tags, want 3", len(tags))
	}
	for i, expected := range []string{"tag1", "tag2", "tag3"} {
		if tags[i].TagName != expected {
			t.Errorf("tags[%d].TagName = %q, want %q", i, tags[i].TagName, expected)
		}
		if tags[i].CategoryName != "armor" {
			t.Errorf("tags[%d].CategoryName = %q", i, tags[i].CategoryName)
		}
		if tags[i].SubcategoryName != "" {
			t.Errorf("tags[%d].SubcategoryName = %q", i, tags[i].SubcategoryName)
		}
	}
}

func TestParseTXT_CommentsAndEmptyLines(t *testing.T) {
	input := "# comment\ntag1\n\ntag2\n  # another comment\ntag3\n  \n"
	r := strings.NewReader(input)
	tags, err := ParseTXT(r, "test", "sub")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Fatalf("got %d tags, want 3", len(tags))
	}
}
