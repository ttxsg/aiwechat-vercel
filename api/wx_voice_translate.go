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
	WECHAT_VOICE_RECOGNIZE_URL = "https://api.weixin.qq.com/cgi-bin/media/voice/addvoicetorecofortext"
)

// Handler 函数是 Vercel 的入口点
func Handler(w http.ResponseWriter, r *http.Request) {
	// 仅处理 POST 请求
	if r.Method != "POST" {
		http.Error(w, "仅支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}
	
	// 解析表单数据
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "解析表单失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 检查文件是否存在
	_, _, err := r.FormFile("voice")
	if err != nil {
		http.Error(w, "未找到语音文件: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 检查参数
	voiceID := r.FormValue("voice_id")
	format := r.FormValue("format")
	if voiceID == "" || format == "" {
		http.Error(w, "必需参数缺失", http.StatusBadRequest)
		return
	}
	
	// 获取 access_token
	appID := os.Getenv("WECHAT_APPID")
	appSecret := os.Getenv("WECHAT_APPSECRET")
	
	if appID == "" || appSecret == "" {
		http.Error(w, "环境变量未配置", http.StatusInternalServerError)
		return
	}
	
	token, err := getAccessToken(appID, appSecret)
	if err != nil {
		http.Error(w, "获取 token 失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 返回成功信息（暂不实现完整语音识别）
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "文件上传成功",
		"token": token,
		"params": map[string]string{
			"voice_id": voiceID,
			"format": format,
		},
	})
}

// 获取微信 access_token
func getAccessToken(appID, appSecret string) (string, error) {
	url := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s",
		WECHAT_TOKEN_URL,
		appID,
		appSecret,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("token request failed: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("parse response failed: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access_token not found")
	}

	return tokenResp.AccessToken, nil
}
