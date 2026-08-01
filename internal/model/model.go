// Package model — структуры golden-датасета каталога (ERD-001).
// Используются генератором (cmd/gen) и валидатором (cmd/validate).
package model

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
