// Command gen генерирует детерминированный golden-датасет каталога.
//
// Детерминизм: одинаковый seed всегда даёт байт-идентичный датасет
// (порядок обхода категорий и офферов фиксирован, случайность — только
// внутри генератора с заданным seed). Строковые seed (BDD-032#S-9,
// например "catalog-2026-08") детерминированно хэшируются в int64 (FNV-1a).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/devcen-online/ecom-golden-dataset/internal/model"
)

const (
	minProducts  = 1000
	unitCount    = 4
	categoryRoot = 1
)

func main() {
	var seedStr string
	var out string
	var products int
	flag.StringVar(&seedStr, "seed", "42", "seed детерминированной генерации (int или строка)")
	flag.IntVar(&products, "products", minProducts, "количество товаров (минимум 1000)")
	flag.StringVar(&out, "out", "dataset.json", "путь к файлу датасета")
	flag.Parse()

	if products < minProducts {
		log.Fatalf("products должен быть >= %d, получено %d", minProducts, products)
	}

	seed := parseSeed(seedStr)
	ds := generate(seed, products)

	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", out, err)
	}
	fmt.Printf("ok: seed %d (%q), %d товаров, %d офферов, %d категорий, %d unit -> %s\n",
		ds.Seed, seedStr, len(ds.Products), len(ds.Offers), len(ds.Categories), unitCount, out)
}

// parseSeed превращает seed-аргумент в int64: целые числа — как есть,
// строки — детерминированным FNV-1a-хэшем (одинакова на всех платформах).
func parseSeed(s string) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

func generate(seed int64, products int) *model.Dataset {
	r := rand.New(rand.NewSource(seed))

	cats := buildCategories(r)
	ds := &model.Dataset{
		Seed:       int(seed),
		Generated:  "deterministic",
		Categories: cats,
		Products:   make([]model.Product, 0, products),
		Offers:     make([]model.Offer, 0, products*minOffers),
	}

	productID := 0
	offerID := 0
	for p := 0; p < products; p++ {
		productID++
		cat := cats[p%len(cats)]

		// Владелец товара меняется по unit-ам, чтобы товары каталога
		// принадлежали нескольким unit (BDD-032#S-8).
		owner := cat.Units[p%len(cat.Units)]

		// Продавцы офферов: владелец + 0..3 других unit (всего 1..4 unit
		// на товар, ERD-001: «от 1 до 4 разных unit»). Индексы
		// (p+i) % len(cat.Units) гарантируют различность unit в наборе.
		extra := 1 + r.Intn(unitCount) // 1..4 — общее число unit на товар
		unitsForProduct := make([]string, 0, extra)
		unitsForProduct = append(unitsForProduct, owner)
		for i := 1; i < extra; i++ {
			unitsForProduct = append(unitsForProduct, cat.Units[(p+i)%len(cat.Units)])
		}

		product := model.Product{
			ID:         productID,
			CategoryID: cat.ID,
			Name:       fmt.Sprintf("Товар %d", productID),
			UnitID:     owner,
			OfferIDs:   make([]int, 0, len(unitsForProduct)),
		}

		for _, u := range unitsForProduct {
			offerID++
			o := model.Offer{
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

const minOffers = 1

var vendors = []string{"VendorA", "VendorB", "VendorC", "VendorD"}

func buildCategories(r *rand.Rand) []model.Category {
	cats := make([]model.Category, 0, 30)
	catID := 0
	next := func() int { catID++; return catID }

	root := model.Category{ID: next(), Name: "Каталог", Units: unitList(0)}
	cats = append(cats, root)

	for i := 0; i < 28; i++ {
		c := model.Category{
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
func DeterministicReport(ds *model.Dataset) string {
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
