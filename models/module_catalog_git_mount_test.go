package models

import "testing"

func TestModuleCatalog_GitHasAMainMount(t *testing.T) {
	entry, ok := CatalogEntry("git")
	if !ok {
		t.Fatal("the catalog has no git")
	}
	if entry.Mount != "main" {
		t.Errorf("git mount = %q, want main", entry.Mount)
	}
}
