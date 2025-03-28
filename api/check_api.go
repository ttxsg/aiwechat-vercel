package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// Response 结构
type Response struct {
	ApiKey    string `json:"apiKey"`
	Endpoint  string `json:"endpoint"`
	ModelName string `json:"modelName"`
	Timestamp string `json:"timestamp"`
}

// Handler 函数 - 符合 Vercel 要求的入口点
func checkAPI(w http.ResponseWriter, r *http.Request) {
	// 设置CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 获取环境变量

	    apiKey := os.Getenv("Github_TORKEN")
	endpoint := os.Getenv("Github_ENDPOINT")
	modelName := os.Getenv("Github_MODEL")
	if apiKey == "" {
		apiKey = "sk-default-test-key" // 默认值，实际应用中移除
	}
	

	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	

	if modelName == "" {
		modelName = "GPT-4o"
	}
	
	// 简单加密API密钥 (Base64编码)
	encodedKey := base64.StdEncoding.EncodeToString([]byte(apiKey))
	
	// 构建响应
	response := Response{
		ApiKey:    encodedKey,
		Endpoint:  endpoint,
		ModelName: modelName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	// 返回JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
