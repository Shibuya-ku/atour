# Task 1 Report: Store 接口、Schema、SQLite Open + Migrate

## Status

**DONE**

## Summary

Implemented `internal/store` package with Store interface, FilterQuery/EventRow types, SQLite/MySQL Open + Migrate, and schema DDL split by driver. Query/import methods stubbed with `not implemented` for later tasks.

## Commits

| SHA | Subject |
|-----|---------|
| `ca0f982` | feat(store): SQLite/MySQL Open 与建表迁移 |

## Files Created

| File | Purpose |
|------|---------|
| `internal/store/store.go` | Store interface, FilterQuery, EventRow, Open() |
| `internal/store/migrate.go` | schemaSQLite / schemaMySQL DDL constants |
| `internal/store/sqlstore.go` | SQLStore: openSQL, Close, Migrate + stub methods |
| `internal/store/store_test.go` | TestOpenMigrateSQLite |

## Dependencies Added

- `modernc.org/sqlite v1.55.0`
- `github.com/go-sql-driver/mysql v1.10.0`

## Test Results

```
go test ./internal/store/ -v
=== RUN   TestOpenMigrateSQLite
--- PASS: TestOpenMigrateSQLite (0.03s)
PASS
ok  	atour/internal/store	0.128s
```

TDD flow followed: test written first (failed with undefined Open), implementation added, test passes.

## Self-Review

### Correctness

- Store interface matches plan: Close, Migrate, ListEvents, ListMatches, ListPlacements, ReplaceAll
- Open accepts `"sqlite"` and `"mysql"`; unsupported drivers return descriptive error
- Migrate selects schemaSQLite or schemaMySQL based on driver
- MySQL schema uses `INT PRIMARY KEY` for events.event_id and `BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY` for matches/placements id columns
- SQLite PRAGMA foreign_keys enabled on open
- Ping on open ensures connection validity

### Scope

- No changes to existing packages (`internal/ajp`, `internal/export`, `cmd/*`)
- List*/ReplaceAll return `not implemented` — intentional for Task 1

### Concerns

- MySQL Migrate not covered by unit test (no MySQL in CI); verified by schema constant review only
- `go get` marked drivers as indirect; acceptable for now, may want direct require in later tasks

## Next Task Readiness

Task 2 can implement ListEvents, ListMatches, ListPlacements, ReplaceAll against the migrated schema.

---

## Review Fix (Important)

**Issue:** MySQL `Migrate` executed entire multi-statement `schemaMySQL` in one `ExecContext`. `go-sql-driver/mysql` rejects multi-statements unless `multiStatements=true`.

**Fix:**
- Refactored `schemaSQLite` / `schemaMySQL` from single multi-statement strings to `[]string` (7 statements each: 3 tables + 4 indexes)
- `Migrate` now executes each DDL statement individually via `ExecContext` for both SQLite and MySQL
- No requirement for users to set `multiStatements=true` in DSN
- Added compile-time assert: `var _ Store = (*SQLStore)(nil)`

## Fix Commit

Subject: `fix(store): Migrate 逐条执行 DDL 以兼容 MySQL`（见 `git log -1`）

## Fix Test Results

```
go test ./internal/store/ -v
=== RUN   TestOpenMigrateSQLite
--- PASS: TestOpenMigrateSQLite (0.03s)
PASS
ok  	atour/internal/store	0.141s
```
