package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/miniprogram/config"
)

// 小程序登录请求结构体
type MiniProgramLoginRequest struct {
	Code string `json:"code"` // 微信登录凭证 code
}

// 小程序登录响应结构体
type MiniProgramLoginResponse struct {
	UserID string `json:"userId"` // 用户唯一标识
	ErrMsg string `json:"errMsg"` // 错误信息
}

// 处理小程序登录
func MiniProgramLogin(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var req MiniProgramLoginRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析请求体失败", http.StatusBadRequest)
		return
	}

	// 初始化微信小程序配置
	wc := wechat.NewWechat()
	memory := cache.NewMemory()
	cfg := &config.Config{
		AppID:     "your-appid",      // 替换为你的小程序 AppID
		AppSecret: "your-appsecret",  // 替换为你的小程序 AppSecret
		Cache:     memory,
	}
	miniProgram := wc.GetMiniProgram(cfg)

	// 调用微信接口，换取 openid 和 session_key
	authResult, err := miniProgram.GetAuth().Code2Session(req.Code)
	if err != nil {
		http.Error(w, "调用微信接口失败", http.StatusInternalServerError)
		return
	}

	if authResult.ErrCode != 0 {
		http.Error(w, fmt.Sprintf("微信接口返回错误: %s", authResult.ErrMsg), http.StatusInternalServerError)
		return
	}

	// 生成 userid（可以根据业务需求自定义）
	userID := generateUserID(authResult.OpenID)

	// 返回 userid 给小程序端
	response := MiniProgramLoginResponse{
		UserID: userID,
		ErrMsg: "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 生成 userid
func generateUserID(openid string) string {
	// 根据 openid 生成 userid（示例）
	return fmt.Sprintf("user_%s", openid[:8])
}
