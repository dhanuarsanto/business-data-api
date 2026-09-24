package usecase

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.internal/business-data-api/internal/domain"
)

type boundEntry struct {
	val int64
	at  time.Time
}

var boundCache sync.Map

const boundTTL = 30 * time.Second

func cacheGet(key string) (int64, bool) {
	if v, ok := boundCache.Load(key); ok {
		e := v.(boundEntry)
		if time.Since(e.at) < boundTTL {
			return e.val, true
		}
		boundCache.Delete(key)
	}
	return 0, false
}

func cacheSet(key string, val int64) {
	boundCache.Store(key, boundEntry{val: val, at: time.Now()})
}

func cacheInvalidatePrefix(prefix string) {
	boundCache.Range(func(k, _ any) bool {
		if strings.HasPrefix(k.(string), prefix) {
			boundCache.Delete(k)
		}
		return true
	})
}

func filterCacheKey(prefix, dbSource, tenant string, limit int, filterID string) string {
	return fmt.Sprintf("%s|%s|%s|%d|%s", prefix, dbSource, tenant, limit, filterID)
}

func fp[E any](p *E) string {
	if p == nil {
		return "0"
	}
	return fmt.Sprintf("%v", *p)
}

func inboxFilterID(f domain.InboxFilter) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		fp(f.StartDate), fp(f.EndDate), fp(f.Terminal), fp(f.Reseller), fp(f.Pengirim), fp(f.Tipe),
		fp(f.Status), f.Pesan, fp(f.RequestFromReseller), fp(f.JawabanFromProvider))
}

func outboxFilterID(f domain.OutboxFilter) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
		fp(f.StartDate), fp(f.EndDate), fp(f.Reseller), fp(f.Penerima), fp(f.Tipe),
		fp(f.Status), f.Pesan, fp(f.ReplyToReseller), fp(f.PerintahProvider))
}
