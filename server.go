package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Server 把 HTTP 处理和 Store 绑在一起。
type Server struct {
	store Store
	// codeLen 短码长度，越长碰撞概率越低。
	codeLen int
	// baseURL 用于把短码拼成完整短链接返回给调用方。
	baseURL string
}

func newServer(store Store, codeLen int, baseURL string) *Server {
	return &Server{store: store, codeLen: codeLen, baseURL: strings.TrimRight(baseURL, "/")}
}

// shorten 处理 POST /api/shorten，body 是 {"url": "..."}。
func (s *Server) shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "body 要像 {\"url\":\"https://...\"}", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		http.Error(w, "url 要以 http:// 或 https:// 开头", http.StatusBadRequest)
		return
	}

	// 碰撞就重生成，最多试 5 次，再撞就认了（内存版几乎不可能）。
	var code string
	var err error
	for i := 0; i < 5; i++ {
		code = genCode(s.codeLen)
		if err = s.store.Save(code, req.URL); err == nil {
			break
		}
	}
	if err != nil {
		http.Error(w, "生成短码失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"code":  code,
		"short": s.baseURL + "/" + code,
		"url":   req.URL,
	})
}

// redirect 处理 GET /{code}，302 跳到原始地址。
func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	// 根路径和 /api 下的不算短码。
	if code == "" || strings.HasPrefix(code, "api") || strings.HasPrefix(code, "favicon") {
		http.NotFound(w, r)
		return
	}
	url, err := s.store.Load(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// Register 把路由挂上。
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/shorten", s.shorten)
	mux.HandleFunc("/", s.redirect)
}

func (s *Server) String() string {
	return fmt.Sprintf("shortlink server codeLen=%d", s.codeLen)
}
