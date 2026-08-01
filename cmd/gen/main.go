// Command gen генерирует детерминированный golden-датасет каталога.
//
// Детерминизм: одинаковый seed всегда даёт байт-идентичный датасет
// (порядок обхода категорий и офферов фиксирован, случайность — только
// внутри генератора с заданным seed).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
)

type Category struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	ParentID *int     `json:"parent_id,omitempty"`
	Units    []string `json:"units"`
}

type Offer struct {
	ID           int      `json:"id"`
	ProductID    int      `json:"product_id"`
	UnitID       string   `json:"unit_id"`
	CategoryID   int      `json:"category_id"`
	Name         string   `json:"name"`
	Price        int      `json:"price"`
	Stock        int      `json:"stock"`
	AggregateVer int      `json:"aggregate_version"`
	SellerID     string   `json:"seller_id"`
	ProductName  string   `json:"product_name"`
	Vendor       string   `json:"vendor"`
	Keywords     []string `json:"keywords"`
}

type Product struct {
	ID         int      `json:"id"`
	CategoryID int      `json:"category_id"`
	Name       string   `json:"name"`
	UnitID     string   `json:"unit_id"`
	OfferIDs   []int    `json:"offer_ids"`
}

type Dataset struct {
	Seed       int        `json:"seed"`
	Generated  string     `json:"generated"`
	Categories []Category `json:"categories"`
	Products   []Product  `json:"products"`
	Offers     []Offer    `json:"offers"`
}

const (
	minProducts  = 1000
	minOffers    = 3
	unitCount    = 4
	categoryRoot = 1
)

func main() {
	var seed int
	var out string
	var products int
	flag.IntVar(&seed, "seed", 42, "seed детерминированной генерации")
	flag.IntVar(&products, "products", minProducts, "количество товаров (минимум 1000)")
	flag.StringVar(&out, "out", "dataset.json", "путь к файлу датасета")
	flag.Parse()

	if products < minProducts {
		log.Fatalf("products должен быть >= %d, получено %d", minProducts, products)
	}

	ds := generate(seed, products)

	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", out, err)
	}
	fmt.Printf("ok: %d товаров, %d офферов, %d категорий, %d unit -> %s\n",
		len(ds.Products), len(ds.Offers), len(ds.Categories), unitCount, out)
}

func generate(seed, products int) *Dataset {
	r := rand.New(rand.NewSource(int64(seed)))

	cats := buildCategories(r)
	ds := &Dataset{
		Seed:       seed,
		Generated:  "deterministic",
		Categories: cats,
		Products:   make([]Product, 0, products),
		Offers:     make([]Offer, 0, products*minOffers),
	}

	productID := 0
	offerID := 0
	for p := 0; p < products; p++ {
		productID++
		cat := cats[p%len(cats)]

		// Собираем продавцов офферов: 1 владелец (unit товара) + от 1 до 3
		// других unit, так что один товар может иметь офферы >= 2 unit
		// (multi-unit/multi-offer кейсы из FR-005).
		unitsForProduct := make([]string, 0, unitCount+2)
		unitsForProduct = append(unitsForProduct, cat.Units[0])
		extra := 1 + r.Intn(3)
		for i := 1; i < extra; i++ {
			unitsForProduct = append(unitsForProduct, cat.Units[i%len(cat.Units)])
		}

		product := Product{
			ID:         productID,
			CategoryID: cat.ID,
			Name:       fmt.Sprintf("Товар %d", productID),
			UnitID:     unitsForProduct[0],
			OfferIDs:   make([]int, 0, len(unitsForProduct)),
		}

		for _, u := range unitsForProduct {
			offerID++
			o := Offer{
				ID:           offerID,
				ProductID:    productID,
				UnitID:       u,
				CategoryID:   cat.ID,
				Name:         fmt.Sprintf("Оффер %d", offerID),
				Price:        100 + r.Intn(50000),
				Stock:        r.Intn(1000),
				AggregateVer: 1,
				SellerID:     "seller-" + u,
				ProductName:  product.Name,
				Vendor:       vendors[r.Intn(len(vendors))],
				Keywords:     []string{product.Name, cat.Name},
			}
			ds.Offers = append(ds.Offers, o)
			product.OfferIDs = append(product.OfferIDs, offerID)
		}
		ds.Products = append(ds.Products, product)
	}

	// Детерминированный порядок: всё уже сгенерировано в фиксированном
	// порядке обхода; сортировки добавляем для устойчивости чтения.
	sort.Slice(ds.Products, func(i, j int) bool { return ds.Products[i].ID < ds.Products[j].ID })
	sort.Slice(ds.Offers, func(i, j int) bool { return ds.Offers[i].ID < ds.Offers[j].ID })
	sort.Slice(ds.Categories, func(i, j int) bool { return ds.Categories[i].ID < ds.Categories[j].ID })
	return ds
}

var vendors = []string{"VendorA", "VendorB", "VendorC", "VendorD"}

func buildCategories(r *rand.Rand) []Category {
	cats := make([]Category, 0, 30)
	catID := 0
	next := func() int { catID++; return catID }

	root := Category{ID: next(), Name: "Каталог", Units: unitList(0)}
	cats = append(cats, root)

	for i := 0; i < 28; i++ {
		c := Category{
			ID:       next(),
			Name:     categoryNames[r.Intn(len(categoryNames))] + fmt.Sprintf(" %d", i+1),
			ParentID: &root.ID,
			Units:    unitList(i),
		}
		cats = append(cats, c)
	}
	return cats
}

func unitList(i int) []string {
	list := make([]string, 0, unitCount)
	for u := 0; u < unitCount; u++ {
		list = append(list, fmt.Sprintf("unit-%d", u))
	}
	return list
}

var categoryNames = strings.Fields("Бытовая техника Электроника Одежда Обувь Аксессуары Косметика Спорт Книги Игрушки Мебель")

// DeterministicReport — вспомогательный отчёт для проверки детерминизма.
func DeterministicReport(ds *Dataset) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "products=%d offers=%d categories=%d\n", len(ds.Products), len(ds.Offers), len(ds.Categories))
	for _, p := range ds.Products {
		ids := make([]string, len(p.OfferIDs))
		for i, id := range p.OfferIDs {
			ids[i] = fmt.Sprintf("%d", id)
		}
		fmt.Fprintf(&sb, "p%d:cat%d:%d:%s\n", p.ID, p.CategoryID, len(p.OfferIDs), strings.Join(ids, ","))
	}
	return sb.String()
}
