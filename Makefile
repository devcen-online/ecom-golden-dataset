SEED ?= 42
PRODUCTS ?= 1000

.PHONY: golden-dataset validate manifest test

golden-dataset: ## детерминированная перегенерация датасета (FR-005, BDD-032#S-8/S-9)
	go run ./cmd/gen -seed "$(SEED)" -products $(PRODUCTS) -out dataset.json

validate: ## валидация датасета по инвариантам ERD-001 (BDD-032#S-8)
	go run ./cmd/validate -in dataset.json

manifest: ## sha256 манифеста датасета (детерминизм, BDD-032#S-9)
	@(shasum -a 256 dataset.json 2>/dev/null || sha256sum dataset.json)

test:
	go test ./...
