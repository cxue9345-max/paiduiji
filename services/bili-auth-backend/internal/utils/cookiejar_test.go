package utils

import (
	"net/http"
	"testing"
)

func TestAddFromSetCookieHeaderAndMerge(t *testing.T) {
	h := http.Header{}
	h.Add("Set-Cookie", "SESSDATA=abc123; Path=/; HttpOnly; Secure")
	h.Add("Set-Cookie", "bili_jct=token; Path=/")

	jar := NewCookieJar()
	jar.AddFromSetCookieHeader(h)
	jar.Merge(map[string]string{"sid": "mysid"})

	if jar.Values["SESSDATA"] != "abc123" {
		t.Fatalf("SESSDATA parse failed")
	}
	if jar.Values["bili_jct"] != "token" || jar.Values["sid"] != "mysid" {
		t.Fatalf("merge/parse failed")
	}
	if !jar.Meta["SESSDATA"].HTTPOnly || !jar.Meta["SESSDATA"].Secure {
		t.Fatalf("cookie meta parse failed")
	}
	if got := jar.CookieString(); got == "" {
		t.Fatalf("cookie string should not be empty")
	}
}

func TestMissingRequiredCookies(t *testing.T) {
	missing := MissingRequiredCookies(map[string]string{"SESSDATA": "1", "sid": "2"})
	if len(missing) != 3 {
		t.Fatalf("expected 3 missing cookies, got %d", len(missing))
	}
}
