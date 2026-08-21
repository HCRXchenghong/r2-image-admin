package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"r2-image-admin/backend/internal/imaging"
)

type aiGenerateRequest struct {
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
}

// handleAIGenerate 代理 OpenAI Images API，生成图片并返回（b64_json 或 url）。
func (s *Server) handleAIGenerate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AI_IMAGE_API_KEY == "" {
		writeError(w, http.StatusBadRequest, "未配置 AI 生图 API Key，请到「设置」中填写")
		return
	}
	if err := s.validateAIEndpoint(s.cfg.AI_IMAGE_API_URL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req aiGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Prompt) == "" || utf8.RuneCountInString(req.Prompt) > 4000 {
		writeError(w, http.StatusBadRequest, "请提供有效的 prompt")
		return
	}
	size := strings.TrimSpace(req.Size)
	if size == "" {
		size = "1024x1024"
	}
	if !allowedAIImageSize(size) {
		writeError(w, http.StatusBadRequest, "不支持的图片尺寸")
		return
	}

	payload := map[string]any{
		"model":           s.cfg.AI_IMAGE_MODEL,
		"prompt":          req.Prompt,
		"n":               1,
		"size":            size,
		"response_format": "b64_json",
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.cfg.AI_IMAGE_API_URL, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "构造请求失败："+err.Error())
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.AI_IMAGE_API_KEY)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "请求 AI 接口失败："+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 24<<20))
	if resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, "AI 生图服务返回错误")
		return
	}
	data, err := sanitizeAIImageResponse(respBody)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI 生图服务返回了无效图片数据")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func allowedAIImageSize(size string) bool {
	switch size {
	case "1024x1024", "1536x1024", "1024x1536":
		return true
	default:
		return false
	}
}

type aiModelsSyncRequest struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

// handleAISyncModels 从上游 OpenAI 兼容接口同步可用模型列表，供设置页选择。
func (s *Server) handleAISyncModels(w http.ResponseWriter, r *http.Request) {
	var req aiModelsSyncRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	apiURL := strings.TrimSpace(req.APIURL)
	if apiURL == "" {
		apiURL = s.cfg.AI_IMAGE_API_URL
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = s.cfg.AI_IMAGE_API_KEY
	}
	if err := s.validateAIEndpoint(apiURL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "请先填写 AI 生图 API Key")
		return
	}

	modelsURL := aiModelsURLFrom(apiURL)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, modelsURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "构造模型同步请求失败："+err.Error())
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "同步模型列表失败："+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, "同步模型列表失败")
		return
	}

	ids := parseAIModelIDs(respBody)
	if len(ids) == 0 {
		writeError(w, http.StatusBadGateway, "上游未返回可用模型，请确认接口地址和 Key 是否正确")
		return
	}
	sort.Strings(ids)
	writeJSON(w, http.StatusOK, map[string]any{"models": ids})
}

// aiModelsURLFrom 根据生图接口地址推导上游模型列表地址。
func aiModelsURLFrom(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		raw = "https://api.openai.com/v1/images/generations"
	}
	if strings.HasSuffix(raw, "/images/generations") {
		return strings.TrimSuffix(raw, "/images/generations") + "/models"
	}
	if strings.HasSuffix(raw, "/models") {
		return raw
	}
	return raw + "/models"
}

// parseAIModelIDs 兼容常见 OpenAI /models 返回结构：{ data: [{ id: "..." }] }。
func parseAIModelIDs(body []byte) []string {
	var obj struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &obj) != nil {
		return nil
	}
	ids := make([]string, 0, len(obj.Data))
	for _, m := range obj.Data {
		if id := strings.TrimSpace(m.ID); validAIModelID(id) {
			ids = append(ids, id)
		}
	}
	return ids
}

func validAIModelID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:/-", r) {
			continue
		}
		return false
	}
	return true
}

func sanitizeAIImageResponse(body []byte) ([]map[string]string, error) {
	var upstream struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &upstream); err != nil || len(upstream.Data) != 1 {
		return nil, io.ErrUnexpectedEOF
	}
	b64 := strings.TrimSpace(upstream.Data[0].B64JSON)
	if len(b64) == 0 || len(b64) > 28<<20 {
		return nil, io.ErrUnexpectedEOF
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(data) == 0 || len(data) > 20<<20 || !imaging.IsRaster(imaging.SniffContentType(data, "")) {
		return nil, io.ErrUnexpectedEOF
	}
	return []map[string]string{{"b64_json": b64}}, nil
}

// validateAIEndpoint 限制管理后台作为网络跳板访问本机或内网地址；可用
// AI_IMAGE_ALLOWED_HOSTS 明确指定受信任的兼容网关主机名。
func (s *Server) validateAIEndpoint(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errInvalidAIEndpoint
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && s.cfg.AppEnv != "production") {
		return errInvalidAIEndpoint
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return errInvalidAIEndpoint
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()) {
		return errInvalidAIEndpoint
	}
	if len(s.cfg.AIAllowedHosts) > 0 {
		for _, allowed := range s.cfg.AIAllowedHosts {
			if host == allowed {
				return nil
			}
		}
		return errInvalidAIEndpoint
	}
	return nil
}

var errInvalidAIEndpoint = &aiEndpointError{}

type aiEndpointError struct{}

func (*aiEndpointError) Error() string {
	return "AI 接口地址不安全或不在允许的主机列表中"
}
