package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShortenAndRedirect(t *testing.T) {
	store := newMemStore()
	srv := newServer(store, 6, "http://short.test")
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 缩短一个 URL
	resp, err := http.Post(ts.URL+"/api/shorten", "application/json", strings.NewReader(`{"url":"https://example.com/very/long/path"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("缩短状态码 %d", resp.StatusCode)
	}
	var out struct {
		Code  string `json:"code"`
		Short string `json:"short"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	t.Logf("POST 返回 code=%q short=%q", out.Code, out.Short)
	if out.Code == "" || !strings.Contains(out.Short, out.Code) {
		t.Fatalf("返回短码异常: %+v", out)
	}

	// 用短码跳回去。自己建 Client 关掉自动跟随重定向，
	// 否则 http.Get 默认会跳到 example.com，那个页面返回 404 会误导测试。
	direct, derr := store.Load(out.Code)
	t.Logf("直接 store.Load(%q) = %q, %v", out.Code, direct, derr)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	get, err := client.Get(ts.URL + "/" + out.Code)
	if err != nil {
		t.Fatal(err)
	}
	defer get.Body.Close()
	if get.StatusCode != http.StatusFound {
		t.Fatalf("跳转状态码应为 302, 得到 %d", get.StatusCode)
	}
	if loc := get.Header.Get("Location"); loc != "https://example.com/very/long/path" {
		t.Fatalf("跳转地址不对: %q", loc)
	}
}

func TestShortenBadURL(t *testing.T) {
	srv := newServer(newMemStore(), 6, "http://x")
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/shorten", "application/json", strings.NewReader(`{"url":"ftp://x"}`))
	if err != nil {
		t.Fatalf("缩短请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法 url 应 400, 得到 %d", resp.StatusCode)
	}
}

func TestHashRingStable(t *testing.T) {
	r := newHashRing(50, "a", "b", "c")
	n1 := r.get("hello")
	n2 := r.get("hello")
	if n1 != n2 {
		t.Fatalf("同一个 key 落到不同节点: %q != %q", n1, n2)
	}
	// 三个节点都要能被分到，不能全挤在一个上。
	seen := map[string]bool{}
	for _, k := range []string{"a1", "b2", "c3", "d4", "e5", "f6", "g7", "h8"} {
		seen[r.get(k)] = true
	}
	if len(seen) < 2 {
		t.Fatalf("分布太集中，只落到 %d 个节点", len(seen))
	}
}

func TestStoreNotFound(t *testing.T) {
	_, err := newMemStore().Load("nope")
	if err != ErrNotFound {
		t.Fatalf("不存在的短码应返回 ErrNotFound, 得到 %v", err)
	}
}
