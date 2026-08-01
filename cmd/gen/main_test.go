package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	multiUnit := 0
	for _, p := range ds.Products {
		if len(p.OfferIDs) < 1 {
			t.Fatalf("товар %d без офферов", p.ID)
		}
		if len(p.OfferIDs) >= 2 {
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

func fmtOffer(o Offer) string {
	b, _ := json.Marshal(o)
	return string(b)
}

func generateAndWrite(t *testing.T, seed, products int, path string) {
	t.Helper()
	data, err := json.MarshalIndent(generate(seed, products), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readDataset(t *testing.T, path string) *Dataset {
	t.Helper()
	ds := &Dataset{}
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
