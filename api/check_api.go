package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func check_api() {
	// 设置路由
	http.HandleFunc("/api/check_api", apiKeyHandler)
	
	// 获取端口，Vercel会自动设置PORT环境变量
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // 默认端口
	}
	
	// 启动服务
	log.Printf("服务启动在 :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// API密钥处理器
func apiKeyHandler(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// 只允许GET请求
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 检索环境变量中的配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	endpoint := os.Getenv("OPENAI_ENDPOINT")
	modelName := os.Getenv("OPENAI_MODEL")
	
	// 验证必要的变量
	if apiKey == "" {
		http.Error(w, "API key not configured", http.StatusInternalServerError)
		return
	}
	
	// 设置默认值
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	if modelName == "" {
		modelName = "GPT-4o" // 默认模型
	}
	
	// 加密API密钥
	encryptedKey, err := encryptApiKey(apiKey)
	if err != nil {
		log.Printf("加密API密钥时出错: %v", err)
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
	
	// 设置JSON响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 编码并发送响应
	json.NewEncoder(w).Encode(response)
}

// 加密API密钥
func encryptApiKey(apiKey string) (string, error) {
	// 从环境变量获取加密密钥，如果不存在则使用默认密钥
	// 警告: 在生产环境中，应该使用安全存储的强密钥
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "default-encryption-key-change-in-production" // 默认密钥
	}
	
	// 创建密钥 (AES-256需要32字节密钥)
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))
	
	// 创建加密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	// 创建随机nonce (一次性数字)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(apiKey), nil)
	
	// Base64编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
