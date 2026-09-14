package i18n_test

import (
	"sort"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
)

func TestDictionaryIsCompleteInBothLanguages(t *testing.T) {
	keys := i18n.Keys()
	if len(keys) == 0 || !sort.StringsAreSorted(keys) {
		t.Fatal("Keys harus terurut dan tidak kosong")
	}
	for _, k := range keys {
		v, ok := i18n.Lookup(k)
		if !ok || v[0] == "" || v[1] == "" {
			t.Errorf("kunci %q tanpa terjemahan lengkap", k)
		}
		if i18n.T("id", k) != v[0] || i18n.T("en", k) != v[1] {
			t.Errorf("T tidak mengembalikan pasangan Lookup untuk %q", k)
		}
	}
	if _, ok := i18n.Lookup("tidak.ada"); ok {
		t.Error("kunci tak dikenal harus mengembalikan false")
	}
	if v, _ := i18n.Lookup("nav.forecast"); v[0] != "Prakiraan Berjalan" {
		t.Errorf("label navigasi prakiraan = %q", v[0])
	}
	if i18n.OtherLang("id") != "en" || i18n.OtherLang("en") != "id" {
		t.Error("OtherLang harus menukar id dan en")
	}
}
