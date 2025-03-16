package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	WECHAT_TOKEN_URL = "https://api.weixin.qq.com/cgi-bin/token"
	WECHAT_VOICE_QUERY_URL = "https://api.weixin.qq.com/cgi-bin/media/voice/queryrecoresultfortext"
)

// Handler 函数是 Vercel 的入口点
func Handler(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	voice_id := r.URL.Query().Get("voice_id")
	lang := r.URL.Query().Get("lang")
	
	if voice_id == "" {
		http.Error(w, "缺少必要参数 voice_id", http.StatusBadRequest)
		return
	}
	
	// 如果没有指定语言，默认使用中文
	if lang == "" {
		lang = "zh_CN"
	}
	
	// 获取微信凭证
	appID := os.Getenv("WECHAT_APPID")
	appSecret := os.Getenv("WECHAT_APPSECRET")
	if appID == "" || appSecret == "" {
		http.Error(w, "微信凭证未配置", http.StatusInternalServerError)
		return
	}

	// 获取access_token
	accessToken, err := getAccessToken(appID, appSecret)
	if err != nil {
		http.Error(w, "获取access_token失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 构建API请求URL
	apiURL := fmt.Sprintf("%s?access_token=%s&voice_id=%s&lang=%s",
		WECHAT_VOICE_QUERY_URL,
		accessToken,
		voice_id,
		lang,
	)
	
	// 发送请求获取语音识别结果
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/json", nil)
	if err != nil {
		http.Error(w, "请求语音识别服务失败: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	
	// 解析响应
	var result struct {
		Result string `json:"result"`
		ErrCode int `json:"errcode"`
		ErrMsg string `json:"errmsg"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, "解析响应失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 检查是否有错误
	if result.ErrCode != 0 && result.ErrMsg != "" {
		http.Error(w, fmt.Sprintf("微信API错误: %d %s", result.ErrCode, result.ErrMsg), http.StatusBadRequest)
		return
	}
	
	// 返回识别结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": result.Result,
	})
}

// 获取微信access_token
func getAccessToken(appID, appSecret string) (string, error) {
	url := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s",
		WECHAT_TOKEN_URL,
		appID,
		appSecret,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("get token request failed: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("parse token response failed: %v", err)
	}
	
	// 检查错误码
	if tokenResp.ErrCode != 0 && tokenResp.ErrMsg != "" {
		return "", fmt.Errorf("wechat error: %d %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access_token not found in response")
	}

	return tokenResp.AccessToken, nil
}
