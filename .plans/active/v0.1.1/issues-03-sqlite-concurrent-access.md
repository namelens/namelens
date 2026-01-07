# v0.1.1 Issue: SQLite "database is locked" under concurrent access

**Priority**: High (blocks batch operations)
**Found**: 2026-01-07 (QA dogfooding session)
**Symptom**: Batch checks fail with SQLite locking errors

## Problem

When running `namelens batch` with multiple names, concurrent goroutines hit SQLite locking issues:

```
Error executing statement: SQLite failure: `database is locked`
```

Affects:
- Rate limit reads: `fetch rate limit: failed to get next row`
- Rate limit writes: `store rate limit: failed to execute query`
- Bootstrap reads: `fetch rdap servers: failed to get next row`

## Reproduction

```bash
# Create batch file with 10+ names
cat > /tmp/names.txt << 'EOF'
arca
cista
pyxis
locu
sigil
fisc
EOF

# Run batch (fails with locking errors)
./bin/namelens batch /tmp/names.txt --profile=startup

# Sequential checks work fine
./bin/namelens check arca --profile=startup
```

## Root Cause Analysis

The libsql store uses a single `*sql.DB` connection. When batch operations spawn concurrent goroutines (one per name in batch), they compete for:
1. Rate limit reads before each check
2. Rate limit writes after each check
3. Bootstrap data reads for RDAP server lookups

SQLite's default mode doesn't handle concurrent writes well without WAL mode or connection pooling.

## Potential Solutions

### Option A: Enable WAL Mode (Quick Fix)
```go
// In store.Open()
_, err = db.Exec("PRAGMA journal_mode=WAL")
```
WAL (Write-Ahead Logging) allows concurrent reads during writes.

### Option B: Serialize Rate Limit Access
Add mutex protection around rate limit store operations:
```go
type Store struct {
    DB *sql.DB
    mu sync.RWMutex
}

func (s *Store) GetRateLimitState(...) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // ...
}

func (s *Store) SetRateLimitState(...) {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ...
}
```

### Option C: Connection Pool Tuning
Configure libsql driver for better concurrency:
```go
db.SetMaxOpenConns(1)  // Force serialization at driver level
```

### Option D: In-Memory Rate Limit Cache
Cache rate limit state in memory, sync to DB periodically or on shutdown.

## Recommendation

**Phase 1 (v0.1.1)**: Option A (WAL mode) + Option C (single conn) - minimal code change
**Phase 2 (v0.1.2)**: Consider Option D for better performance at scale

## Workaround

Sequential checks work fine. Users can run checks one at a time:
```bash
for name in arca ferrata kybos; do
  ./bin/namelens check "$name" --profile=startup
done
```

## Related

- `01-rate-limit-reset.md` - Rate limit admin tooling (works fine for single-threaded access)
- Libsql docs: https://docs.turso.tech/sdk/go/reference

---
*Found during QA dogfooding of fulmen-secrets name generation, 2026-01-07*
