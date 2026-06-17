package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"

	"github.com/leandrofars/oktopus/internal/usp/usp_msg"
	"github.com/leandrofars/oktopus/internal/usp/usp_utils"
	"github.com/leandrofars/oktopus/internal/utils"
	"github.com/nats-io/nats.go/jetstream"
)

// sanitizeKVKey replaces characters that are invalid in NATS KV keys.
// NATS KV keys are NATS subjects: alphanumeric, dash, underscore, dot only.
func sanitizeKVKey(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
			b.WriteRune(c)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

// paramCacheKey builds a stable NATS KV key for a USP GET request.
// Format: usp.<sanitized_sn>.<sha256_hex_of_sorted_paths>
func paramCacheKey(sn string, paths []string) string {
	sorted := make([]string, len(paths))
	copy(sorted, paths)
	sort.Strings(sorted)
	h := sha256.Sum256([]byte(strings.Join(sorted, ",")))
	return "usp." + sanitizeKVKey(sn) + "." + hex.EncodeToString(h[:])
}

// isValidGetResp returns true only when the response body looks like a USP GetResp
// (contains "req_path_results"). This guards against caching USP Error responses
// that the controller happens to write with HTTP 200.
func isValidGetResp(body []byte) bool {
	return bytes.Contains(body, []byte("req_path_results"))
}

// purgeDeviceParamCache removes all cached GET entries for a device from the KV store.
// Called after a successful SET so stale values are not served.
func purgeDeviceParamCache(ctx context.Context, kv jetstream.KeyValue, sn string) {
	if kv == nil {
		return
	}
	prefix := "usp." + sanitizeKVKey(sn) + "."
	lister, err := kv.ListKeys(ctx)
	if err != nil {
		log.Printf("paramCache: ListKeys error for device %s: %v", sn, err)
		return
	}
	var toDelete []string
	for key := range lister.Keys() {
		if strings.HasPrefix(key, prefix) {
			toDelete = append(toDelete, key)
		}
	}
	for _, key := range toDelete {
		if err := kv.Delete(ctx, key); err != nil {
			log.Printf("paramCache: delete key %s error: %v", key, err)
		}
	}
}

// deviceGetCachedMsg is like deviceGetMsg but checks the NATS KV cache first.
// On cache miss it performs a live GET and stores the result for future requests.
// The existing /get endpoint is left untouched (always live) so DevicesDiscovery
// and other callers are not affected.
func (a *Api) deviceGetCachedMsg(w http.ResponseWriter, r *http.Request) {
	sn := getSerialNumberFromRequest(r)
	mtp, err := getMtpFromRequest(r, w)
	if err != nil {
		return
	}
	if mtp == "" {
		var ok bool
		mtp, ok = deviceStateOK(w, a.nc, sn)
		if !ok {
			return
		}
	}

	var get usp_msg.Get
	utils.MarshallDecoder(&get, r.Body)

	// --- cache read ---
	if a.paramKv != nil && len(get.ParamPaths) > 0 {
		cacheKey := paramCacheKey(sn, get.ParamPaths)
		if entry, err := a.paramKv.Get(r.Context(), cacheKey); err == nil {
			w.Header().Set("X-Cache", "HIT")
			w.Write(entry.Value())
			return
		}
	}

	// --- cache miss: live GET via response recorder ---
	rec := httptest.NewRecorder()
	msg := usp_utils.NewGetMsg(get)
	if err := sendUspMsg(msg, sn, rec, a.nc, mtp); err != nil {
		// sendUspMsg already wrote the error; copy it out
		for k, vs := range rec.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
		return
	}

	body := rec.Body.Bytes()

	// --- cache write (only on a genuine GetResp, not a USP Error) ---
	if a.paramKv != nil && rec.Code == http.StatusOK && isValidGetResp(body) && len(get.ParamPaths) > 0 {
		cacheKey := paramCacheKey(sn, get.ParamPaths)
		if _, putErr := a.paramKv.Put(r.Context(), cacheKey, body); putErr != nil {
			log.Printf("paramCache: failed to store key for device %s: %v", sn, putErr)
		}
	}

	// --- forward recorder response to real writer ---
	for k, vs := range rec.Header() {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if rec.Code != http.StatusOK {
		w.WriteHeader(rec.Code)
	}
	w.Write(body)
}
