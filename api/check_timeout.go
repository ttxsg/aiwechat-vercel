package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// 请求结构
type AuthRequest struct {
	Code string `json:"code"`
}

// 响应结构
type AuthResponse struct {
	Valid        bool   `json:"valid"`
	Message      string `json:"message,omitempty"`
	ExpiryDate   string `json:"expiry_date,omitempty"`
	DaysRemaining int   `json:"days_remaining,omitempty"`
}

// 授权码数据库（示例，实际应使用数据库存储）
var authCodes = map[string]time.Time{
	"ABC123-DEF456": time.Now().AddDate(0, 1, 0),  // 1个月后过期
	"TEST-CODE-999": time.Now().AddDate(0, 3, 0),  // 3个月后过期
	"TRIAL-VERSION": time.Now().AddDate(0, 0, 7),  // 7天后过期
}

func main() {
	http.HandleFunc("/api/check_timeout", handleAuthCheck)
	log.Println("授权验证服务器运行在 :8080...")
	http.ListenAndServe(":8080", nil)
}

func handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	// 只允许POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证授权码
	expiryDate, exists := authCodes[req.Code]
	
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 构建响应
	response := AuthResponse{}
	
	if !exists {
		response.Valid = false
		response.Message = "无效的授权码"
	} else if expiryDate.Before(time.Now()) {
		response.Valid = false
		response.Message = "授权码已过期"
	} else {
		response.Valid = true
		response.ExpiryDate = expiryDate.Format("2006-01-02T15:04:05Z")
		
		// 计算剩余天数
		daysRemaining := int(expiryDate.Sub(time.Now()).Hours() / 24)
		response.DaysRemaining = daysRemaining
		response.Message = "授权有效"
	}
	
	// 返回JSON响应
	json.NewEncoder(w).Encode(response)
}
