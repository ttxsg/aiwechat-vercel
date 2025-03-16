package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
	"log"
	"github.com/pwh-pwh/aiwechat-vercel/chat"
	"github.com/pwh-pwh/aiwechat-vercel/config"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"
)

const (
	WECHAT_VOICE_RECOGNIZE_URL = "https://api.weixin.qq.com/cgi-bin/media/voice/addvoicetorecofortext"
	WECHAT_TOKEN_URL            = "https://api.weixin.qq.com/cgi-bin/token"
)

// 语音识别请求参数结构
type VoiceRecognizeRequest struct {
	AccessToken string `json:"access_token" form:"access_token" binding:"required"`
	VoiceID    string `json:"voice_id" form:"voice_id" binding:"required"`
	Format     string `json:"format" form:"format" binding:"required"`
	Lang       string `json:"lang" form:"lang" binding:"omitempty"`
}

// 语音识别响应结构
type VoiceRecognizeResponse struct {
	ErrCode int    `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
	Result  struct {
		Text       string `json:"text"`
		Sentence   []struct {
			Text      string `json:"text"`
		CONFidence float64 `json:"confidence"`
		} `json:"sentence"`
	} `json:"result,omitempty"`
}

func init() {
	http.HandleFunc("/api/wechat/voice/upload", handleVoiceUpload)
}

func handleVoiceUpload(w http.ResponseWriter, r *http.Request) {
	// 获取微信凭证
	appID := os.Getenv("WECHAT_APPID")
	appSecret := os.Getenv("WECHAT_APPSECRET")
	if appID == "" || appSecret == "" {
		log.Println("微信凭证未配置")
		http.Error(w, "微信凭证未配置", http.StatusInternalServerError)
		return
	}

	// 获取access_token
	accessToken, err := getAccessToken(appID, appSecret)
	if err != nil {
		log.Printf("获取access_token失败: %v", err)
		http.Error(w, "无法获取微信访问凭证", http.StatusInternalServerError)
		return
	}

	// 解析请求参数和文件
	form := multipart.NewForm()
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("文件上传解析失败: %v", err)
		http.Error(w, "文件上传失败", http.StatusBadRequest)
		return
	}

	var req VoiceRecognizeRequest
	if err := form.Decode(&req); err != nil {
		log.Printf("参数解析失败: %v", err)
		http.Error(w, "参数格式错误", http.StatusBadRequest)
		return
	}

	// 补全必填参数
	req.AccessToken = accessToken

	// 构建API请求
	apiURL := fmt.Sprintf("%s?access_token=%s&voice_id=%s&format=%s&lang=%s",
		WECHAT_VOICE_RECOGNIZE_URL,
		req.AccessToken,
		req.VoiceID,
		req.Format,
		strings.TrimSpace(req.Lang),
	)

	// 创建请求体
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// 添加语音文件
	file, handler, err := r.FormFile("voice")
	if err != nil {
		log.Printf("获取语音文件失败: %v", err)
		http.Error(w, "未找到语音文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := writer.AddFormFile("voice", handler.Filename, file); err != nil {
		log.Printf("文件添加失败: %v", err)
		http.Error(w, "文件上传失败", http.StatusInternalServerError)
		return
	}

	if err := writer.Close(); err != nil {
		log.Printf("构建表单失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 发送HTTP请求
	client := &http.Client{Timeout: 10 * time.Second}
	reqHTTP, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	reqHTTP.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(reqHTTP)
	if err != nil {
		log.Printf("API调用失败: %v", err)
		http.Error(w, "语音识别服务不可用", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 解析响应
	var respData VoiceRecognizeResponse
	if err := json.Unmarshal(respBody, &respData); err != nil {
		log.Printf("解析响应失败: %v", err)
		http.Error(w, "无效的响应格式", http.StatusBadRequest)
		return
	}

	// 处理微信API错误
	if respData.ErrCode != 0 {
		log.Printf("微信API错误: %d %s", respData.ErrCode, respData.ErrMsg)
		http.Error(w, fmt.Sprintf("识别失败：%d %s", respData.ErrCode, respData.ErrMsg), http.StatusBadRequest)
		return
	}

	// 构建返回结果
	result := struct {
		Text      string `json:"text"`
		Sentence  []struct {
			Text      string     `json:"text"`
			Confidence float64  `json:"confidence"`
		} `json:"sentence"`
	}{
		Text:      respData.Result.Text,
		Sentence:  respData.Result.Sentence,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get token failed: status code %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn  int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("parse token response failed: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access_token not found in response")
	}

	return tokenResp.AccessToken, nil
}
