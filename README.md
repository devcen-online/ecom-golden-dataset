# ecom-golden-dataset

Детерминированный генератор golden-датасета каталога (подзадача #32, FR-005).

## Требования (FR-005)
- ≥ 1000 товаров;
- multi-unit/multi-offer кейсы: один товар имеет офферы от ≥ 2 unit;
- детерминизм: одинаковый seed → байт-идентичный датасет;
- регенерация одной командой.

## Использование

```bash
go build ./cmd/gen
./gen -seed 42 -products 1000 -out dataset.json
./gen -seed 42 -products 100000 -out dataset-big.json   # увеличенный датасет для load-тестов
```

Повторный запуск с тем же `-seed` и теми же параметрами даёт идентичный файл.

## Проверка

```bash
go test ./...
```

Тесты: байт-идентичность при повторе seed (TestDeterministicRegeneration),
разные seed → разные датасеты (TestDifferentSeedDifferentOutput),
форма датасета: ≥1000 товаров, multi-unit присутствует, нет дублей офферов (TestDatasetShape).

## Формат

JSON: `{seed, generated, categories[], products[], offers[]}`.
- `products[].offer_ids` — офферы товара; товар принадлежит `unit_id`,
  офферы могут быть от других unit (multi-seller кейс);
- `offers[].unit_id` — тенант-владелец оффера, `seller_id` — продавец,
  `aggregate_version` — версия агрегата для тестов projection (не откатывать
  read model устаревшими событиями).

## Ссылки
- PRD: docs/prd/PRD-032-test-infrastructure.md, FR-005; BDD S-8..S-13.
- Данные для генерации датасета реального масштаба: docs/data/gold/seo_ekb.xml (полный YML-фид ~713 МБ), docs/data/example/feed-example.xml.
