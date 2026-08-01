// Command validate проверяет golden-датасет каталога на инварианты ERD-001
// (INV-1..INV-7) — BDD-032#S-8: «датасет валидируется скриптом без ошибок».
//
//	validate -in dataset.json
//
// Exit 0 при отсутствии нарушений, 1 — при любом нарушении.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/devcen-online/ecom-golden-dataset/internal/model"
)

const minProducts = 1000

// Validate проверяет инварианты ERD-001 и возвращает список нарушений
// (пустой список — датасет валиден).
func Validate(ds *model.Dataset) []string {
	var v []string
	bad := func(format string, args ...any) { v = append(v, fmt.Sprintf(format, args...)) }

	// INV-1: generated фиксирован (не время), детерминизм проверяет генератор.
	if ds.Generated != "deterministic" {
		bad("dataset.generated должен быть \"deterministic\", есть %q (INV-1)", ds.Generated)
	}

	// INV-2: объём.
	if len(ds.Products) < minProducts {
		bad("товаров %d < %d (INV-2)", len(ds.Products), minProducts)
	}

	// Офферы по товарам (порядок датасета) и категориям.
	offersByProduct := map[int][]*model.Offer{}
	catIDs := map[int]bool{}
	for _, c := range ds.Categories {
		catIDs[c.ID] = true
	}
	for i := range ds.Offers {
		o := &ds.Offers[i]
		offersByProduct[o.ProductID] = append(offersByProduct[o.ProductID], o)
	}

	// INV-3: товары нескольких unit; есть товары с офферами от >= 2 unit;
	// первый оффер — unit-владелец.
	ownerUnits := map[string]bool{}
	for _, p := range ds.Products {
		ownerUnits[p.UnitID] = true
	}
	if len(ownerUnits) < 2 {
		bad("товары принадлежат только одному unit: %v (INV-3)", ownerUnits)
	}
	multiUnitProducts := 0
	for _, p := range ds.Products {
		units := map[string]bool{}
		for _, o := range offersByProduct[p.ID] {
			units[o.UnitID] = true
		}
		if len(units) >= 2 {
			multiUnitProducts++
		}
		if len(p.OfferIDs) > 0 {
			first := firstOffer(ds, p.OfferIDs[0])
			if first != nil && first.UnitID != p.UnitID {
				bad("товар %d: первый оффер %d принадлежит %s, а не владельцу %s (INV-3)",
					p.ID, p.OfferIDs[0], first.UnitID, p.UnitID)
			}
		}
	}
	if multiUnitProducts == 0 {
		bad("нет товаров с офферами от >= 2 unit (INV-3)")
	}

	// INV-4: ссылочная целостность и порядок offer_ids.
	prodIDs := map[int]bool{}
	for _, p := range ds.Products {
		prodIDs[p.ID] = true
		if !catIDs[p.CategoryID] {
			bad("товар %d ссылается на несуществующую категорию %d (INV-4)", p.ID, p.CategoryID)
		}
	}
	offerIDs := map[int]bool{}
	for _, o := range ds.Offers {
		offerIDs[o.ID] = true
		if !prodIDs[o.ProductID] {
			bad("оффер %d ссылается на несуществующий товар %d (INV-4)", o.ID, o.ProductID)
		}
		if !catIDs[o.CategoryID] {
			bad("оффер %d ссылается на несуществующую категорию %d (INV-4)", o.ID, o.CategoryID)
		}
	}
	for _, p := range ds.Products {
		got := make([]int, 0, len(offersByProduct[p.ID]))
		for _, o := range offersByProduct[p.ID] {
			got = append(got, o.ID)
		}
		if !slices.Equal(got, p.OfferIDs) {
			bad("товар %d: offer_ids не совпадают с набором офферов (INV-4)", p.ID)
		}
		for _, oid := range p.OfferIDs {
			if !offerIDs[oid] {
				bad("товар %d: offer_id %d не существует (INV-4)", p.ID, oid)
			}
		}
	}

	// INV-5: счётчики; ID уникальны и возрастают.
	total := 0
	for _, p := range ds.Products {
		total += len(p.OfferIDs)
	}
	if total != len(ds.Offers) {
		bad("сумма offer_ids (%d) != len(offers) (%d) (INV-5)", total, len(ds.Offers))
	}
	for i := 1; i < len(ds.Products); i++ {
		if ds.Products[i].ID <= ds.Products[i-1].ID {
			bad("ID товаров не возрастают (INV-5)")
			break
		}
	}
	for i := 1; i < len(ds.Offers); i++ {
		if ds.Offers[i].ID <= ds.Offers[i-1].ID {
			bad("ID офферов не возрастают (INV-5)")
			break
		}
	}

	// INV-6: aggregate_version >= 1 и монотонен по (product_id, unit_id).
	lastVer := map[string]int{}
	for _, o := range ds.Offers {
		if o.AggregateVer < 1 {
			bad("оффер %d: aggregate_version %d < 1 (INV-6)", o.ID, o.AggregateVer)
		}
		key := fmt.Sprintf("%d/%s", o.ProductID, o.UnitID)
		if prev, ok := lastVer[key]; ok && o.AggregateVer < prev {
			bad("оффер %d: aggregate_version %d < предыдущего %d для %s (INV-6)", o.ID, o.AggregateVer, prev, key)
		}
		lastVer[key] = o.AggregateVer
	}

	// INV-7: seller_id выводится из unit_id; check-ограничения ERD (price/stock >= 0).
	for _, o := range ds.Offers {
		if want := "seller-" + o.UnitID; o.SellerID != want {
			bad("оффер %d: seller_id %q != %q (INV-7)", o.ID, o.SellerID, want)
		}
		if o.Price < 0 {
			bad("оффер %d: price %d < 0 (check)", o.ID, o.Price)
		}
		if o.Stock < 0 {
			bad("оффер %d: stock %d < 0 (check)", o.ID, o.Stock)
		}
	}
	return v
}

func firstOffer(ds *model.Dataset, id int) *model.Offer {
	for i := range ds.Offers {
		if ds.Offers[i].ID == id {
			return &ds.Offers[i]
		}
	}
	return nil
}

func main() {
	var in string
	flag.StringVar(&in, "in", "dataset.json", "путь к файлу датасета")
	flag.Parse()

	data, err := os.ReadFile(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read %s: %v\n", in, err)
		os.Exit(1)
	}
	var ds model.Dataset
	if err := json.Unmarshal(data, &ds); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse %s: %v\n", in, err)
		os.Exit(1)
	}

	issues := Validate(&ds)
	if len(issues) > 0 {
		for _, i := range issues {
			fmt.Fprintln(os.Stderr, "invalid:", i)
		}
		fmt.Printf("%s: %d нарушений\n", in, len(issues))
		os.Exit(1)
	}
	fmt.Printf("ok: %s (%d товаров, %d офферов, %d категорий)\n",
		in, len(ds.Products), len(ds.Offers), len(ds.Categories))
}
