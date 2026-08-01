package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devcen-online/ecom-golden-dataset/internal/model"
)

func loadDataset(t *testing.T, path string) *model.Dataset {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ds model.Dataset
	if err := json.Unmarshal(b, &ds); err != nil {
		t.Fatal(err)
	}
	return &ds
}

// TestValidateRealDataset — поставляемый dataset.json (генератор с seed 42)
// проходит все инварианты ERD-001 (BDD-032#S-8: «датасет валидируется
// скриптом без ошибок»).
func TestValidateRealDataset(t *testing.T) {
	ds := loadDataset(t, filepath.Join("..", "..", "dataset.json"))
	issues := Validate(ds)
	if len(issues) > 0 {
		t.Fatalf("dataset.json невалиден:\n%s", strings.Join(issues, "\n"))
	}
	if len(ds.Products) < 1000 {
		t.Fatalf("товаров должно быть >= 1000, есть %d", len(ds.Products))
	}
}

func TestValidateCorruptedDataset(t *testing.T) {
	base := loadDataset(t, filepath.Join("..", "..", "dataset.json"))

	t.Run("удалён оффер", func(t *testing.T) {
		ds := *base
		ds.Offers = ds.Offers[1:]
		issues := Validate(&ds)
		if !hasViolation(issues, "INV-4") && !hasViolation(issues, "INV-5") {
			t.Fatalf("удаление оффера должно нарушить INV-4/INV-5: %v", issues)
		}
	})

	t.Run("неверный seller_id", func(t *testing.T) {
		ds := *base
		ds.Offers[0].SellerID = "seller-other-unit"
		issues := Validate(&ds)
		if !hasViolation(issues, "INV-7") {
			t.Fatalf("неверный seller_id должен нарушить INV-7: %v", issues)
		}
	})

	t.Run("первый оффер не от unit-владельца", func(t *testing.T) {
		ds := *base
		firstProduct := ds.Products[0]
		if len(firstProduct.OfferIDs) == 0 {
			t.Skip("товар без офферов")
		}
		o := firstOffer(&ds, firstProduct.OfferIDs[0])
		o.UnitID = "unit-other"
		issues := Validate(&ds)
		if !hasViolation(issues, "INV-3") {
			t.Fatalf("первый оффер чужого unit должен нарушить INV-3: %v", issues)
		}
	})

	t.Run("мало товаров", func(t *testing.T) {
		ds := *base
		ds.Products = ds.Products[:10]
		issues := Validate(&ds)
		if !hasViolation(issues, "INV-2") {
			t.Fatalf("мало товаров должно нарушить INV-2: %v", issues)
		}
	})
}

func TestValidateSmallCustomDataset(t *testing.T) {
	ds := &model.Dataset{
		Generated: "deterministic",
		Categories: []model.Category{{ID: 1, Name: "root", Units: []string{"unit-0", "unit-1"}}},
		Products:   []model.Product{{ID: 1, CategoryID: 1, UnitID: "unit-0", OfferIDs: []int{1}}},
		Offers: []model.Offer{{
			ID: 1, ProductID: 1, CategoryID: 1, UnitID: "unit-0",
			SellerID: "seller-unit-0", Price: 100, Stock: 1, AggregateVer: 1,
		}},
	}
	issues := Validate(ds)
	// Для маленького датасета допустимы нарушения объёма (INV-2)
	// и отсутствия multi-unit товаров (INV-3).
	for _, i := range issues {
		if !strings.Contains(i, "INV-2") && !strings.Contains(i, "INV-3") {
			t.Errorf("неожиданное нарушение: %s", i)
		}
	}
}

func hasViolation(issues []string, inv string) bool {
	for _, i := range issues {
		if strings.Contains(i, inv) {
			return true
		}
	}
	return false
}
