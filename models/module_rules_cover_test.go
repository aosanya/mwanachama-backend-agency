package models

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k, on := range set {
		if on {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func TestModuleRequires_ListsTheDeclaredDependenciesAndNothingForUnknownCodes(t *testing.T) {
	if got := ModuleRequires("forms"); !reflect.DeepEqual(got, []string{"catalog", "git"}) {
		t.Errorf("forms requires %v, want catalog and git", got)
	}
	if got := ModuleRequires("auth"); len(got) != 0 {
		t.Errorf("auth requires %v, want nothing", got)
	}
	if got := ModuleRequires("no-such-module"); got != nil {
		t.Errorf("an unknown module requires %v, want nil", got)
	}
}

func TestWithRequiredModules_ClosesTheEnabledSetOverRequires(t *testing.T) {
	got := sortedKeys(WithRequiredModules(map[string]bool{"forms": true}))
	want := []string{"catalog", "forms", "git"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("closure of forms = %v, want %v", got, want)
	}
}

func TestWithRequiredModules_IgnoresModulesSwitchedOffAndDoesNotMutateTheInput(t *testing.T) {
	input := map[string]bool{"forms": false, "voting": true}
	got := sortedKeys(WithRequiredModules(input))
	if !reflect.DeepEqual(got, []string{"voting"}) {
		t.Errorf("closure = %v, want only voting", got)
	}
	if len(input) != 2 || input["forms"] || !input["voting"] {
		t.Errorf("input was changed to %v", input)
	}
}

func TestWithRequiredModules_EmptyInputGivesAnEmptyNonNilSet(t *testing.T) {
	got := WithRequiredModules(nil)
	if got == nil || len(got) != 0 {
		t.Errorf("closure of nothing = %v, want an empty set", got)
	}
}

func TestModulesNeededBySteps_NamesEveryModuleWhoseCatalogListsAPlayedKind(t *testing.T) {
	got := sortedKeys(ModulesNeededBySteps(map[StepKind]bool{StepKindNotify: true, StepKindFillForm: true}))
	want := []string{"comm", "forms"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("needed = %v, want %v", got, want)
	}
}

func TestModulesNeededBySteps_AKindMarkedFalseOrUnlistedNeedsNothing(t *testing.T) {
	needed := ModulesNeededBySteps(map[StepKind]bool{StepKindChat: false, StepKindPost: true})
	if len(needed) != 0 {
		t.Errorf("needed = %v, want none", needed)
	}
	if len(ModulesNeededBySteps(nil)) != 0 {
		t.Error("no step kinds must need no module")
	}
}

func TestParseModuleMounts_BlankIsNoMountsAndBadJSONIsRefused(t *testing.T) {
	for _, blank := range []string{"", "   ", "\n\t"} {
		mounts, err := ParseModuleMounts(blank)
		if mounts != nil || err != nil {
			t.Errorf("blank %q gave %v, %v", blank, mounts, err)
		}
	}
	if _, err := ParseModuleMounts("{not json"); !errors.Is(err, ErrInvalidMounts) {
		t.Errorf("bad json error = %v, want ErrInvalidMounts", err)
	}
	if _, err := ParseModuleMounts(`{"name":"x"}`); !errors.Is(err, ErrInvalidMounts) {
		t.Errorf("an object where a list belongs error = %v, want ErrInvalidMounts", err)
	}
}

func TestParseModuleMounts_ReadsNamesKeysAndIcons(t *testing.T) {
	mounts, err := ParseModuleMounts(`[{"name":"Barn","key":"barn","icon":"wheat"},{"name":"Yard"}]`)
	if err != nil {
		t.Fatal(err)
	}
	want := []ModuleMount{{Name: "Barn", Key: "barn", Icon: "wheat"}, {Name: "Yard"}}
	if !reflect.DeepEqual(mounts, want) {
		t.Errorf("mounts = %+v, want %+v", mounts, want)
	}
}

func TestEffectiveModuleMounts_StoredMountsWinOverTheCatalogDefault(t *testing.T) {
	got := EffectiveModuleMounts("assetmanager", `[{"name":"Barn","key":"barn"}]`)
	if len(got) != 1 || got[0].Key != "barn" {
		t.Errorf("mounts = %+v, want the stored barn", got)
	}
}

func TestEffectiveModuleMounts_FallsBackToTheCatalogDefaultForBlankEmptyOrBrokenStorage(t *testing.T) {
	for _, stored := range []string{"", "[]", "garbage"} {
		got := EffectiveModuleMounts("assetmanager", stored)
		if len(got) != 1 || got[0].Key != "assets" {
			t.Errorf("stored %q gave %+v, want the default assets mount", stored, got)
		}
	}
}

func TestEffectiveModuleMounts_UnknownModuleAndMountlessModuleHaveNone(t *testing.T) {
	if got := EffectiveModuleMounts("no-such-module", ""); got != nil {
		t.Errorf("unknown module mounts = %+v, want nil", got)
	}
	if got := EffectiveModuleMounts("custody", ""); got != nil {
		t.Errorf("custody has no mount but gave %+v", got)
	}
}

func TestMaxMountsFor_MountableAllowsTheCapSingleMountAllowsOneOthersNone(t *testing.T) {
	cases := map[string]int{"assetmanager": MaxModuleMounts, "auth": 1, "custody": 0}
	for code, want := range cases {
		entry, ok := CatalogEntry(code)
		if !ok {
			t.Fatalf("no catalog entry for %s", code)
		}
		if got := MaxMountsFor(entry); got != want {
			t.Errorf("%s max mounts = %d, want %d", code, got, want)
		}
	}
}

func TestValidateModuleMounts_AcceptsEmptyAndWellFormedMounts(t *testing.T) {
	if err := ValidateModuleMounts("custody", nil); err != nil {
		t.Errorf("no mounts on a mountless module = %v, want nil", err)
	}
	ok := []ModuleMount{{Name: "Barn", Key: "barn", Icon: "wheat"}, {Name: "Yard", Key: "yard"}}
	if err := ValidateModuleMounts("assetmanager", ok); err != nil {
		t.Errorf("valid mounts refused: %v", err)
	}
}

func TestValidateModuleMounts_RefusalsAreAllInvalidMounts(t *testing.T) {
	tooMany := make([]ModuleMount, MaxModuleMounts+1)
	for i := range tooMany {
		tooMany[i] = ModuleMount{Name: strings.Repeat("n", i+1), Key: strings.Repeat("k", i+1)}
	}
	cases := []struct {
		name   string
		code   string
		mounts []ModuleMount
	}{
		{"unknown module", "no-such-module", []ModuleMount{{Name: "A", Key: "a"}}},
		{"module without a mount", "custody", []ModuleMount{{Name: "A", Key: "a"}}},
		{"two on a single-mount module", "auth", []ModuleMount{{Name: "A", Key: "a"}, {Name: "B", Key: "b"}}},
		{"over the cap", "assetmanager", tooMany},
		{"blank name", "assetmanager", []ModuleMount{{Name: "", Key: "a"}}},
		{"whitespace only name", "assetmanager", []ModuleMount{{Name: "  ", Key: "a"}}},
		{"leading space", "assetmanager", []ModuleMount{{Name: " A", Key: "a"}}},
		{"trailing space", "assetmanager", []ModuleMount{{Name: "A ", Key: "a"}}},
		{"name over the length cap", "assetmanager", []ModuleMount{{Name: strings.Repeat("é", MaxMountNameLength+1), Key: "a"}}},
		{"control character", "assetmanager", []ModuleMount{{Name: "A\x07B", Key: "a"}}},
		{"names equal when folded", "assetmanager", []ModuleMount{{Name: "Barn", Key: "a"}, {Name: "bARN", Key: "b"}}},
		{"blank key", "assetmanager", []ModuleMount{{Name: "A", Key: ""}}},
		{"repeated key", "assetmanager", []ModuleMount{{Name: "A", Key: "k"}, {Name: "B", Key: "k"}}},
		{"icon outside the set", "assetmanager", []ModuleMount{{Name: "A", Key: "a", Icon: "rocket"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := ValidateModuleMounts(c.code, c.mounts); !errors.Is(err, ErrInvalidMounts) {
				t.Errorf("error = %v, want ErrInvalidMounts", err)
			}
		})
	}
}

func TestValidateModuleMounts_ANameOfExactlyTheCapInRunesIsAccepted(t *testing.T) {
	name := strings.Repeat("é", MaxMountNameLength)
	if err := ValidateModuleMounts("assetmanager", []ModuleMount{{Name: name, Key: "a"}}); err != nil {
		t.Errorf("a %d-rune name was refused: %v", MaxMountNameLength, err)
	}
}

func TestMountKeyFromName_KeepsLowercaseLettersAndLaterDigitsOnly(t *testing.T) {
	cases := map[string]string{
		"Cattle Barn": "cattlebarn",
		"Yard 2":      "yard2",
		"2nd Yard":    "ndyard",
		"A-B_C!":      "abc",
		"123":         "mount",
		"":            "mount",
		"---":         "mount",
		"Zoë":         "zo",
	}
	for name, want := range cases {
		if got := MountKeyFromName(name); got != want {
			t.Errorf("MountKeyFromName(%q) = %q, want %q", name, got, want)
		}
	}
}
