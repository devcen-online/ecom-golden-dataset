package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/devcen-online/ecom-golden-dataset/internal/model"
)

func TestDeterministicRegeneration(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "d1.json")
	generateAndWrite(t, 42, 1000, out)
	d1 := readDataset(t, out)

	out2 := filepath.Join(dir, "d2.json")
	generateAndWrite(t, 42, 1000, out2)
	d2 := readDataset(t, out2)

	if !bytes.Equal(mustRead(t, out), mustRead(t, out2)) {
		t.Fatal("одинаковый seed должен давать байт-идентичный датасет")
	}
	if DeterministicReport(d1) != DeterministicReport(d2) {
		t.Fatal("отчёты должны совпадать")
	}
}

// TestStringSeedDeterministic — BDD-032#S-9: строковый seed
// ("catalog-2026-08") хэшируется детерминированно.
func TestStringSeedDeterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	generateAndWriteStr(t, "catalog-2026-08", 1000, a)
	generateAndWriteStr(t, "catalog-2026-08", 1000, b)
	if !bytes.Equal(mustRead(t, a), mustRead(t, b)) {
		t.Fatal("одинаковый строковый seed должен давать байт-идентичный датасет")
	}
}

func TestParseSeed(t *testing.T) {
	if parseSeed("42") != 42 {
		t.Fatal("числовой seed должен использоваться как есть")
	}
	a := parseSeed("catalog-2026-08")
	b := parseSeed("catalog-2026-08")
	if a != b {
		t.Fatalf("строковый seed должен хэшироваться детерминированно: %d != %d", a, b)
	}
	if parseSeed("catalog-2026-08") == parseSeed("catalog-2026-09") {
		t.Fatal("разные строковые seed должны хэшироваться по-разному")
	}
}

func TestDifferentSeedDifferentOutput(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	generateAndWrite(t, 1, 1000, a)
	generateAndWrite(t, 2, 1000, b)
	if bytes.Equal(mustRead(t, a), mustRead(t, b)) {
		t.Fatal("разные seed должны давать разные датасеты")
	}
}

func TestDatasetShape(t *testing.T) {
	ds := generate(7, 1000)
	if len(ds.Products) < 1000 {
		t.Fatalf("товаров должно быть >= 1000, есть %d", len(ds.Products))
	}

	// BDD-032#S-8: товары принадлежат нескольким unit.
	ownerUnits := map[string]bool{}
	for _, p := range ds.Products {
		ownerUnits[p.UnitID] = true
		if len(p.OfferIDs) < 1 {
			t.Fatalf("товар %d без офферов", p.ID)
		}
	}
	if len(ownerUnits) < 2 {
		t.Fatalf("товары должны принадлежать нескольким unit, есть только %v", ownerUnits)
	}

	// FR-005: хотя бы один товар с офферами от >= 2 разных unit.
	multiUnit := 0
	for _, p := range ds.Products {
		units := map[string]bool{}
		for _, oid := range p.OfferIDs {
			for _, o := range ds.Offers {
				if o.ID == oid {
					units[o.UnitID] = true
					break
				}
			}
		}
		if len(units) >= 2 {
			multiUnit++
		}
	}
	if multiUnit == 0 {
		t.Fatal("нет ни одного multi-unit товара (требование FR-005)")
	}

	seen := map[string]bool{}
	for _, o := range ds.Offers {
		if seen[fmtOffer(o)] {
			t.Fatalf("дублирующийся оффер %v", o)
		}
		seen[fmtOffer(o)] = true
	}
}

func fmtOffer(o model.Offer) string {
	b, _ := json.Marshal(o)
	return string(b)
}

func generateAndWrite(t *testing.T, seed int64, products int, path string) {
	t.Helper()
	data, err := json.MarshalIndent(generate(seed, products), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func generateAndWriteStr(t *testing.T, seed string, products int, path string) {
	t.Helper()
	data, err := json.MarshalIndent(generate(parseSeed(seed), products), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readDataset(t *testing.T, path string) *model.Dataset {
	t.Helper()
	ds := &model.Dataset{}
	b := mustRead(t, path)
	if err := json.Unmarshal(b, ds); err != nil {
		t.Fatal(err)
	}
	return ds
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
