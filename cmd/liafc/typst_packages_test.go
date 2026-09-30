package main

import (
	"slices"
	"testing"
)

func TestDocFontTags(t *testing.T) {
	tags, err := docFontTags("inter, math")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tags, []string{"liaf_doc_sem_serif", "liaf_doc_sem_mono"}) {
		t.Fatalf("tags = %v", tags)
	}
	if tags, _ := docFontTags(""); tags != nil {
		t.Fatalf("vazio deveria manter todas, veio %v", tags)
	}
	if _, err := docFontTags("comic-sans"); err == nil {
		t.Fatal("família desconhecida foi aceita")
	}
}
