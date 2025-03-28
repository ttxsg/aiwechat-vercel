package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// AuthRequest 定义请求结构
type AuthRequest struct {
	Code string `json:"code"`
}

// AuthResponse 定义响应结构
type AuthResponse struct {
	Valid         bool   `json:"valid"`
	Message       string `json:"message,omitempty"`
	ExpiryDate    string `json:"expiry_date,omitempty"`
	DaysRemaining int    `json:"days_remaining,omitempty"`
}

// 演示用的授权码数据
// 实际应用中应使用数据库或其他持久存储
var authCodes = map[string]time.Time{
	"ABC123-DEF456": time.Now().AddDate(0, 1, 0),  // 1个月后过期
	"TEST-CODE-999": time.Now().AddDate(0, 3, 0),  // 3个月后过期
	"TRIAL-VERSION": time.Now().AddDate(0, 0, 7),  // 7天后过期
}

// CheckTimeout 处理授权验证请求
// 这个函数名称必须是Handler或以Handler结尾，按照Vercel的约定
func CheckTimeout(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头，允许跨域请求
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	// 处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	
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

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 验证授权码
	expiryDate, exists := authCodes[req.Code]
	
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
