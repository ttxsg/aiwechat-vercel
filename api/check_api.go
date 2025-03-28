package handler

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
func Handler(w http.ResponseWriter, r *http.Request) {
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
package api

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"
)

// ConfigResponse 定义API响应结构
type ConfigResponse struct {
    ApiKey    string `json:"apiKey"`
    Endpoint  string `json:"endpoint"`
    ModelName string `json:"modelName"`
    Timestamp string `json:"timestamp"`
}

// Handler 处理 API 请求
func Handler(w http.ResponseWriter, r *http.Request) {
    // CORS 设置
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    
    // 处理预检请求
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    // 只允许 GET 请求
    if r.Method != "GET" && r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // 获取环境变量
    apiKey := os.Getenv("Github_TORKEN")
	endpoint := os.Getenv("Github_ENDPOINT")
	modelName := os.Getenv("Github_MODEL")
    
    if apiKey == "" {
        http.Error(w, "API key not configured", http.StatusInternalServerError)
        return
    }
    
    // 设置默认值
    if endpoint == "" {
        endpoint = "https://api.openai.com/v1"
    }
    if modelName == "" {
        modelName = "GPT-4o"
    }
    
    // 加密 API 密钥
    encryptedKey, err := encryptApiKey(apiKey)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    // 构建响应
    response := ConfigResponse{
        ApiKey:    encryptedKey,
        Endpoint:  endpoint,
        ModelName: modelName,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
    
    // 返回 JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// 加密 API 密钥
func encryptApiKey(apiKey string) (string, error) {
    // 简化的加密实现
    encryptionKey := os.Getenv("ENCRYPTION_KEY")
    if encryptionKey == "" {
        encryptionKey = "default-encryption-key-change-in-production"
    }
    
    key := make([]byte, 32)
    copy(key, []byte(encryptionKey))
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(apiKey), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
