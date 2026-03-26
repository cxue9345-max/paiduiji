package utils

import (
	"bili-auth-backend/internal/model"
	"net/http"
	"sort"
	"strings"
)

type CookieJar struct {
	Values map[string]string
	Meta   map[string]model.CookieMeta
}

func NewCookieJar() *CookieJar {
	return &CookieJar{Values: map[string]string{}, Meta: map[string]model.CookieMeta{}}
}

func (j *CookieJar) AddFromSetCookieHeader(header http.Header) {
	for _, raw := range header.Values("Set-Cookie") {
		parts := strings.Split(raw, ";")
		if len(parts) == 0 {
			continue
		}
		kv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		if len(kv) != 2 || kv[0] == "" {
			continue
		}
		key, val := kv[0], kv[1]
		j.Values[key] = val
		meta := model.CookieMeta{Raw: raw}
		for _, attr := range parts[1:] {
			a := strings.TrimSpace(attr)
			if a == "HttpOnly" {
				meta.HTTPOnly = true
			}
			if a == "Secure" {
				meta.Secure = true
			}
			if strings.HasPrefix(strings.ToLower(a), "path=") {
				meta.Path = strings.TrimPrefix(a, "Path=")
			}
			if strings.HasPrefix(strings.ToLower(a), "domain=") {
				meta.Domain = strings.TrimPrefix(a, "Domain=")
			}
		}
		j.Meta[key] = meta
	}
}

func (j *CookieJar) Merge(values map[string]string) {
	for k, v := range values {
		j.Values[k] = v
	}
}

func (j *CookieJar) CookieString() string {
	keys := make([]string, 0, len(j.Values))
	for k := range j.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, k := range keys {
		items = append(items, k+"="+j.Values[k])
	}
	return strings.Join(items, "; ")
}

func MissingRequiredCookies(m map[string]string) []string {
	required := []string{"SESSDATA", "bili_jct", "DedeUserID", "DedeUserID__ckMd5", "sid"}
	missing := make([]string, 0)
	for _, k := range required {
		if _, ok := m[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}
