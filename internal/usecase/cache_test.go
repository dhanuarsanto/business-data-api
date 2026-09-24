package usecase

import (
	"testing"
	"time"
)

func TestCacheInvalidatePrefix(t *testing.T) {
	boundCache.Store("inbox|postgres|maxtop|500|1", boundEntry{val: 10, at: time.Now()})
	boundCache.Store("inbox|postgres|toplink|500|1", boundEntry{val: 20, at: time.Now()})
	boundCache.Store("inbox|mssql|maxtop|500|1", boundEntry{val: 30, at: time.Now()})
	boundCache.Store("outbox|postgres|maxtop|500|1", boundEntry{val: 40, at: time.Now()})

	cacheInvalidatePrefix("inbox|postgres|maxtop")

	_, inboxMaxtop := cacheGet("inbox|postgres|maxtop|500|1")
	_, inboxToplink := cacheGet("inbox|postgres|toplink|500|1")
	_, inboxMS := cacheGet("inbox|mssql|maxtop|500|1")
	_, outboxPG := cacheGet("outbox|postgres|maxtop|500|1")

	if inboxMaxtop {
		t.Fatal("kunci inbox|postgres|maxtop harus ikut invalidated")
	}
	if !inboxToplink || !inboxMS || !outboxPG {
		t.Fatal("kunci prefiks lain harus tetap ada")
	}

	boundCache.Delete("inbox|postgres|toplink|500|1")
	boundCache.Delete("inbox|mssql|maxtop|500|1")
	boundCache.Delete("outbox|postgres|maxtop|500|1")
}
