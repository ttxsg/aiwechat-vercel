

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	Gemini_Welcome_Reply_Key = "geminiWelcomeReply"
	Gemini_Key               = "geminiKey"
	NOTION_API_VERSION = "2022-06-28"
)
type UserConfig struct {
	UserId         string `json:"用户id"`
	NOTION_API_KEY string `json:"NOTION_API_KEY"`
	DATABASE_ID    string `json:"DATABASE_ID"`
	DatabaseType     string `json:"数据库类型"` // 0: 记账数据库, 1: 记工资入账数据库
}
func GetGeminiKey() string {
	return os.Getenv(Gemini_Key)
}

func Wx(rw http.ResponseWriter, req *http.Request) {
	wc := wechat.NewWechat()
	memory := cache.NewMemory()
	cfg := &offConfig.Config{
		AppID:     "",
		AppSecret: "",
		Token:     config.GetWxToken(),
		Cache:     memory,
	}
	officialAccount := wc.GetOfficialAccount(cfg)

	// 传入 request 和 responseWriter
	server := officialAccount.GetServer(req, rw)
	server.SkipValidate(true)

	// 设置接收消息的处理方法
	server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
		// 回复消息：演示回复用户发送的消息
		replyMsg := handleWxMessage(msg)
		text := message.NewText(replyMsg)
		return &message.Reply{MsgType: message.MsgTypeText, MsgData: text}
	})

	// 处理消息接收以及回复
	err := server.Serve()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 发送回复的消息
	server.Send()
}

func handleWxMessage(msg *message.MixMessage) (replyMsg string) {
	msgType := msg.MsgType
	msgContent := msg.Content
	userId := string(msg.FromUserName)
	var Msg_get string // 定义 Msg_get 变量

	// 判断消息类型是否是文本消息
	if msgType == message.MsgTypeText {
		// 检查文本消息是否以 "删除记账账号" 开头
		if strings.HasPrefix(msgContent, "删除记账账号") {
			// 删除记账账号
			feedback := deleteFromNotionConfig(userId, "0") // 数据库类型为 0（记账数据库）
			// 输出反馈信息
			var replyBuilder strings.Builder
			for _, message := range feedback {
				log.Println(message)
				replyBuilder.WriteString(message + "\n")
			}
			replyMsg = replyBuilder.String()
			return
		}

		// 检查文本消息是否以 "添加记账账号" 开头
		if strings.HasPrefix(msgContent, "添加记账账号") {
			// 解析消息内容，提取 NOTION_API_KEY 和 DATABASE_ID
			lines := strings.Split(msgContent, "\n")
			if len(lines) < 3 {
				replyMsg = "格式错误，请按照以下格式输入：\n添加记账账号\nNOTION_API_KEY = 'your_api_key'\nDATABASE_ID = 'your_database_id'"
				return
			}

			// 提取 NOTION_API_KEY 和 DATABASE_ID
			notionApiKey := strings.TrimSpace(strings.Split(lines[1], "=")[1])
			notionApiKey = strings.Trim(notionApiKey, "'\"")
			databaseId := strings.TrimSpace(strings.Split(lines[2], "=")[1])
			databaseId = strings.Trim(databaseId, "'\"")

			// 插入到 Notion 配置数据库，数据库类型默认为 0（记账数据库）
			feedback := insertToNotionConfig(userId, notionApiKey, databaseId, "0")

			// 输出反馈信息
			var replyBuilder strings.Builder
			for _, message := range feedback {
				log.Println(message)
				replyBuilder.WriteString(message + "\n")
			}
			replyMsg = replyBuilder.String()
			return
		}

		// 检查文本消息是否以 "添加工资记账" 开头
		if strings.HasPrefix(msgContent, "添加工资记账") {
			// 解析消息内容，提取 NOTION_API_KEY 和 DATABASE_ID
			lines := strings.Split(msgContent, "\n")
			if len(lines) < 3 {
				replyMsg = "格式错误，请按照以下格式输入：\n添加记账账号\nNOTION_API_KEY = 'your_api_key'\nDATABASE_ID = 'your_database_id'"
				return
			}

			// 提取 NOTION_API_KEY 和 DATABASE_ID
			notionApiKey := strings.TrimSpace(strings.Split(lines[1], "=")[1])
			notionApiKey = strings.Trim(notionApiKey, "'\"")
			databaseId := strings.TrimSpace(strings.Split(lines[2], "=")[1])
			databaseId = strings.Trim(databaseId, "'\"")

			// 插入到 Notion 配置数据库，数据库类型默认为 1（工资数据库）
			feedback := insertToNotionConfig(userId, notionApiKey, databaseId, "1")

			// 输出反馈信息
			var replyBuilder strings.Builder
			for _, message := range feedback {
				log.Println(message)
				replyBuilder.WriteString(message + "\n")
			}
			replyMsg = replyBuilder.String()
			return
		}

		// 检查文本消息以 "1 " 开头
		if len(msgContent) >= 2 && msgContent[:2] == "1 " {
			// 查询用户配置，数据库类型为 1（工资数据库）
			userConfig, err := QueryUserConfig(userId, "1")
			if err != nil {
				log.Println("Error querying user config:", err)
				replyMsg = "用户未绑定，请先绑定账号和 Notion 数据库"
				return
			}

			Msg_get = msgContent[2:] // 去掉前面的 "1 " 进行处理
			log.Println("Msg_get:", Msg_get)

			// 进行 API 调用，替换 data_send 为 Msg_get
			expenses, err := processRequest_gongzi(Msg_get)
			if err != nil {
				log.Println("Error processing request:", err)
				replyMsg = "调用 processRequest_gongzi 失败: " + err.Error()
				return
			}

			// 将 expenses 转换为 JSON 字符串
			expensesJson, err := json.Marshal(expenses)
			if err != nil {
				log.Println("Error marshalling expenses to JSON:", err)
				replyMsg = "调用转换为 JSON 失败: " + err.Error()
				return
			}
			replyMsg = string(expensesJson)

			// 调用 Notion API 插入数据
			feedback := insertToNotion_gongzi(userConfig.DATABASE_ID, userConfig.NOTION_API_KEY, expenses)

			// 输出反馈信息
			var replyBuilder strings.Builder
			for _, message := range feedback {
				log.Println(message)
				replyBuilder.WriteString(message + "\n")
			}
			replyMsg = replyBuilder.String()
			return
		}

		// 检查文本消息是否以 "0 " 开头
		if len(msgContent) >= 2 && msgContent[:2] == "0 " {
			// 查询用户配置，数据库类型为 0（记账数据库）
			userConfig, err := QueryUserConfig(userId, "0")
			if err != nil {
				log.Println("Error querying user config:", err)
				replyMsg = "用户未绑定，请先绑定账号和 Notion 数据库"
				return
			}

			Msg_get = msgContent[2:] // 去掉前面的 "0 " 进行处理
			log.Println("Msg_get:", Msg_get)

			// 进行 API 调用，替换 data_send 为 Msg_get
			expenses, err := processRequest(Msg_get)
			if err != nil {
				log.Println("Error processing request:", err)
				replyMsg = "调用 processRequest 失败: " + err.Error()
				return
			}

			// 将 expenses 转换为 JSON 字符串
			expensesJson, err := json.Marshal(expenses)
			if err != nil {
				log.Println("Error marshalling expenses to JSON:", err)
				replyMsg = "调用转换为 JSON 失败: " + err.Error()
				return
			}
			replyMsg = string(expensesJson)

			// 调用 Notion API 插入数据
			feedback := insertToNotion(userConfig.DATABASE_ID, userConfig.NOTION_API_KEY, expenses)

			// 输出反馈信息
			var replyBuilder strings.Builder
			for _, message := range feedback {
				log.Println(message)
				replyBuilder.WriteString(message + "\n")
			}
			replyMsg = replyBuilder.String()
			return
		}

		// 如果不是以 "0 " 或 "1 " 开头，则使用正常的聊天处理
		bot := chat.GetChatBot(config.GetUserBotType(userId))
		replyMsg = bot.Chat(userId, msgContent)
	} else {
		// 如果是其他类型的消息，使用媒体消息的处理逻辑
		bot := chat.GetChatBot(config.GetUserBotType(userId))
		replyMsg = bot.HandleMediaMsg(msg)
	}
	return
}

// processRequest_gongzi 调用 AI 接口，生成工资记录的 JSON 数据
func processRequest_gongzi(Msg_get string) ([]map[string]interface{}, error) {
	log.Println("Msg_get:", Msg_get)
	todayDate := time.Now().Format("2006-01-02")
	log.Println("Today's date:", todayDate)

	apiKey := GetGeminiKey()
	log.Println("apiKey:", apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is empty")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

	// 构造请求数据
	prompt := fmt.Sprintf(`
		今天是 %s，请根据以下工资或收入记录生成 JSON 数据：
		%s
		如果没有指定日期，默认使用今天；如果没有金额，请估算一个合理的数值；单位默认是公司单位（例如 ***公司）。；标签从下面选择：固定工资 ，兼职写代码 ，家人资助，人情往来礼金，其他兼职；
		返回的 JSON 格式如下：
		[
			{
				"名称": "2025年1月份工资",
				"标签": ["工资"],
				"日期": "2025-01-12",
				"金额": 10000,
				"单位": "***"
			}
		]
		支持一次性处理多条工资记录，请确保返回的数据是 JSON 格式，不要包含无关内容或注释。
	`, todayDate, Msg_get)

	data := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
				},
			},
		},
	}

	log.Println("发送的请求 data:", data)
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling data: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Gemini POST 请求 resp:", resp)
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	log.Println("Gemini API response body:", string(body))

	var apiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	if len(apiResponse.Candidates) == 0 || len(apiResponse.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no valid response from Gemini API")
	}

	jsonText := apiResponse.Candidates[0].Content.Parts[0].Text
	jsonText = strings.TrimSpace(jsonText)
	jsonText = strings.TrimPrefix(jsonText, "```json")
	jsonText = strings.TrimSuffix(jsonText, "```")

	log.Println("Cleaned JSON text:", jsonText)

	var expenses []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonText), &expenses); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON content: %v", err)
	}

	// 校验返回的数据格式
	for _, expense := range expenses {
		if _, ok := expense["名称"]; !ok {
			return nil, fmt.Errorf("missing required field: 名称")
		}
		if _, ok := expense["标签"]; !ok {
			return nil, fmt.Errorf("missing required field: 标签")
		}
		if _, ok := expense["日期"]; !ok {
			return nil, fmt.Errorf("missing required field: 日期")
		}
		if _, ok := expense["金额"]; !ok {
			return nil, fmt.Errorf("missing required field: 金额")
		}
		if _, ok := expense["单位"]; !ok {
			// 如果单位字段缺失，默认设置为公司单位
			return nil, fmt.Errorf("missing required field: 单位")
		}
	}

	return expenses, nil
}
// insertToNotion_gongzi 将工资记录插入到 Notion 数据库
// insertToNotion_gongzi 将工资记录插入到 Notion 数据库
func insertToNotion_gongzi(databaseId, notionApiKey string, expenses []map[string]interface{}) []string {
	log.Println("expenses:", expenses)

	// 设置请求头
	headers := map[string]string{
		"Authorization":  fmt.Sprintf("Bearer %s", notionApiKey),
		"Content-Type":   "application/json",
		"Notion-Version": NOTION_API_VERSION,
	}

	var feedback []string

	// 确保 expenses 是一个 []map[string]interface{}
	// 如果 expenses 是嵌套的数组，先将其展平
	var flatExpenses []map[string]interface{}
	for _, expense := range expenses {
		if nestedExpenses, ok := expense["data"].([]map[string]interface{}); ok {
			// 如果 expense 包含嵌套的 "data" 字段
			flatExpenses = append(flatExpenses, nestedExpenses...)
		} else {
			// 否则直接添加到 flatExpenses
			flatExpenses = append(flatExpenses, expense)
		}
	}

	// 向 Notion 插入每条记录
	for _, entry := range flatExpenses {
		// 处理多选字段 "标签"
		var tags []map[string]interface{}
		if tagList, ok := entry["标签"].([]interface{}); ok {
			for _, tag := range tagList {
				tags = append(tags, map[string]interface{}{"name": tag})
			}
		} else {
			// 如果标签不是列表，直接添加
			tags = append(tags, map[string]interface{}{"name": entry["标签"]})
		}

		payload := map[string]interface{}{
			"parent": map[string]interface{}{
				"database_id": databaseId,
			},
			"properties": map[string]interface{}{
				"名称": map[string]interface{}{
					"title": []map[string]interface{}{
						{
							"text": map[string]interface{}{
								"content": entry["名称"].(string), // 确保字段名称和数据类型正确
							},
						},
					},
				},
				"标签": map[string]interface{}{
					"multi_select": tags, // 多选字段
				},
				"日期": map[string]interface{}{
					"date": map[string]interface{}{
						"start": entry["日期"].(string), // 确保字段名称和数据类型正确
					},
				},
				"金额": map[string]interface{}{
					"number": entry["金额"].(float64), // 确保字段名称和数据类型正确
				},
				"工作单位": map[string]interface{}{ // 工作单位字段
					"select": map[string]interface{}{
						"name": entry["单位"].(string), // 直接使用 entry["单位"] 的值
					},
				},
			},
		}

		// 打印请求的 JSON 数据（用于调试）
		payloadBytes, _ := json.Marshal(payload)
		log.Println("Notion API request payload:", string(payloadBytes))

		// 发送请求插入数据
		req, err := http.NewRequest("POST", "https://api.notion.com/v1/pages", bytes.NewBuffer(payloadBytes))
		if err != nil {
			feedback = append(feedback, fmt.Sprintf("Error creating request for %v: %v", entry["名称"], err))
			continue
		}

		for key, value := range headers {
			req.Header.Add(key, value)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			feedback = append(feedback, fmt.Sprintf("Error inserting %v: %v", entry["名称"], err))
			continue
		}
		defer resp.Body.Close()

		// 打印 Notion API 的响应（用于调试）
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("Notion API response:", string(body))

		if resp.StatusCode == http.StatusOK {
			feedback = append(feedback, fmt.Sprintf("Successfully added: %v", entry["名称"]))
		} else {
			feedback = append(feedback, fmt.Sprintf("Failed to add %v: %v", entry["名称"], resp.Status))
		}
	}

	return feedback
}
// deleteFromNotionConfig 删除 Notion 配置数据库中的记录
func deleteFromNotionConfig(userId, databaseType string) []string {
	// Notion 配置数据库的 Database ID
	configDatabaseId := os.Getenv("NOTION_CONFIG_DATABASE_ID")
	if configDatabaseId == "" {
		return []string{"NOTION_CONFIG_DATABASE_ID 未设置"}
	}

	// Notion API Key
	notionApiKey := os.Getenv("NOTION_API_KEY")
	if notionApiKey == "" {
		return []string{"NOTION_API_KEY 未设置"}
	}

	// 查询数据库中是否已存在相同的用户 ID 和数据库类型
	existingPageId, err := queryExistingPageId(userId, databaseType, configDatabaseId, notionApiKey)
	if err != nil {
		return []string{fmt.Sprintf("Error querying existing page: %v", err)}
	}

	if existingPageId == "" {
		return []string{fmt.Sprintf("未找到用户 %s 的记账账号", userId)}
	}

	// 设置请求头
	headers := map[string]string{
		"Authorization":  fmt.Sprintf("Bearer %s", notionApiKey),
		"Content-Type":   "application/json",
		"Notion-Version": NOTION_API_VERSION,
	}

	// 构造请求数据，将记录标记为已归档（archived）
	payload := map[string]interface{}{
		"archived": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return []string{fmt.Sprintf("Error marshalling payload: %v", err)}
	}

	// 发送请求更新数据（标记为已归档）
	url := fmt.Sprintf("https://api.notion.com/v1/pages/%s", existingPageId)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return []string{fmt.Sprintf("Error creating request: %v", err)}
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{fmt.Sprintf("Error sending request: %v", err)}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{fmt.Sprintf("Error reading response: %v", err)}
	}

	if resp.StatusCode == http.StatusOK {
		return []string{fmt.Sprintf("Successfully deleted: %s", userId)}
	} else {
		return []string{fmt.Sprintf("Failed to delete %s: %s", userId, string(body))}
	}
}
// QueryUserConfig 查询 Notion 数据库，获取用户的配置
func QueryUserConfig(userId, databaseType string) (*UserConfig, error) {
	// Notion 配置数据库的 Database ID
	configDatabaseId := os.Getenv("NOTION_CONFIG_DATABASE_ID")
	if configDatabaseId == "" {
		return nil, fmt.Errorf("NOTION_CONFIG_DATABASE_ID 未设置")
	}

	// Notion API Key
	notionApiKey := os.Getenv("NOTION_API_KEY")
	if notionApiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY 未设置")
	}

	// 构造查询请求
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", configDatabaseId)
	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"and": []map[string]interface{}{
				{
					"property": "用户id",
					"title": map[string]interface{}{
						"equals": userId,
					},
				},
				{
					"property": "数据库类型",
					"select": map[string]interface{}{
						"equals": databaseType, // 0: 记账数据库, 1: 记工资入账数据库
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", notionApiKey))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Notion-Version", NOTION_API_VERSION)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	// 提取用户配置
	results := result["results"].([]interface{})
	if len(results) == 0 {
		return nil, fmt.Errorf("user %s with database type %s not found in Notion config database", userId, databaseType)
	}

	firstResult := results[0].(map[string]interface{})
	properties := firstResult["properties"].(map[string]interface{})

	userIdField := properties["用户id"].(map[string]interface{})
	notionApiKeyField := properties["NOTION_API_KEY"].(map[string]interface{})
	databaseIdField := properties["DATABASE_ID"].(map[string]interface{})
	databaseTypeField := properties["数据库类型"].(map[string]interface{})

	userConfig := &UserConfig{
		UserId:         userIdField["title"].([]interface{})[0].(map[string]interface{})["plain_text"].(string),
		NOTION_API_KEY: notionApiKeyField["rich_text"].([]interface{})[0].(map[string]interface{})["plain_text"].(string),
		DATABASE_ID:    databaseIdField["rich_text"].([]interface{})[0].(map[string]interface{})["plain_text"].(string),
		DatabaseType:     databaseTypeField["select"].(map[string]interface{})["name"].(string),
	}

	return userConfig, nil
}

// insertToNotionConfig 将用户配置插入到 Notion 配置数据库
func insertToNotionConfig(userId, notionApiKey, databaseId, databaseType string) []string {
	// Notion 配置数据库的 Database ID
	configDatabaseId := os.Getenv("NOTION_CONFIG_DATABASE_ID")
	if configDatabaseId == "" {
		return []string{"NOTION_CONFIG_DATABASE_ID 未设置"}
	}

	// Notion API Key
	notionApiKeyForConfig := os.Getenv("NOTION_API_KEY")
	if notionApiKeyForConfig == "" {
		return []string{"NOTION_API_KEY 未设置"}
	}

	// 设置请求头
	headers := map[string]string{
		"Authorization":  fmt.Sprintf("Bearer %s", notionApiKeyForConfig),
		"Content-Type":   "application/json",
		"Notion-Version": NOTION_API_VERSION,
	}

	// 1. 查询是否已存在相同的用户 ID 和数据库类型
	existingPageId, err := queryExistingPageId(userId, databaseType, configDatabaseId, notionApiKeyForConfig)
	if err != nil {
		return []string{fmt.Sprintf("Error querying existing page: %v", err)}
	}

	// 2. 构造请求数据
	payload := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": configDatabaseId,
		},
		"properties": map[string]interface{}{
			"用户id": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"text": map[string]interface{}{
							"content": userId,
						},
					},
				},
			},
			"NOTION_API_KEY": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"text": map[string]interface{}{
							"content": notionApiKey,
						},
					},
				},
			},
			"DATABASE_ID": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"text": map[string]interface{}{
							"content": databaseId,
						},
					},
				},
			},
			"数据库类型": map[string]interface{}{
				"select": map[string]interface{}{
					"name": databaseType, // 0: 记账数据库, 1: 记工资入账数据库
				},
			},
		},
	}

	// 3. 发送请求插入或更新数据
	var url string
	if existingPageId != "" {
		// 如果记录已存在，更新数据
		url = fmt.Sprintf("https://api.notion.com/v1/pages/%s", existingPageId)
	} else {
		// 如果记录不存在，插入新数据
		url = "https://api.notion.com/v1/pages"
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return []string{fmt.Sprintf("Error marshalling payload: %v", err)}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return []string{fmt.Sprintf("Error creating request: %v", err)}
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{fmt.Sprintf("Error sending request: %v", err)}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{fmt.Sprintf("Error reading response: %v", err)}
	}

	if resp.StatusCode == http.StatusOK {
		if existingPageId != "" {
			return []string{fmt.Sprintf("Successfully updated: %s", userId)}
		} else {
			return []string{fmt.Sprintf("Successfully added: %s", userId)}
		}
	} else {
		return []string{fmt.Sprintf("Failed to add/update %s: %s", userId, string(body))}
	}
}

// queryExistingPageId 查询数据库中是否已存在相同的用户 ID 和数据库类型
func queryExistingPageId(userId, databaseType, configDatabaseId, notionApiKey string) (string, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", configDatabaseId)
	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"and": []map[string]interface{}{
				{
					"property": "用户id",
					"title": map[string]interface{}{
						"equals": userId,
					},
				},
				{
					"property": "数据库类型",
					"select": map[string]interface{}{
						"equals": databaseType,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", notionApiKey))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Notion-Version", NOTION_API_VERSION)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	// 提取已有的记录 ID
	results := result["results"].([]interface{})
	if len(results) > 0 {
		firstResult := results[0].(map[string]interface{})
		return firstResult["id"].(string), nil
	}

	return "", nil
}
// func processRequest(Msg_get string) ([]map[string]interface{}, error) {
//     log.Println("Msg_get:", Msg_get)
//     todayDate := time.Now().Format("2006-01-02")
//     log.Println("Today's date:", todayDate)

//     apiKey := GetGeminiKey()
//     log.Println("apiKey:", apiKey)
//     if apiKey == "" {
//         return nil, fmt.Errorf("Gemini API key is empty")
//     }

//     url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

   
// // 	data := map[string]interface{}{
// //     "contents": []map[string]interface{}{
// //         {
// //             "parts": []map[string]interface{}{
// //                 {
// //                     "text": fmt.Sprintf("今天是 %s ，记账 %s ，如果没有指定时间，默认是今天；如果没有金额，帮我虚拟估算一个数；支付方式只有 支付宝 或微信 或银行卡 ；标签从以下内容选 生活吃喝加买菜 房贷-银行金 医疗保健 水电物业 出行 家人-互动生活穿衣用品 家用设备 电子设备 电话费 旅游 其他 摩托车 网购 学习课程；开支类型从下面选择：其他 日常开支 固定开支 社交娱乐开支 节假日开支 教育和自我提升开支 医疗保健开支 意外或紧急开支!! 交通开支(出行) 加油 购物；注意 你给我返回一个json组合成的列表格式，不要和内容无关的东西，不要重复给我，其中不需要换行符，下面给你一个例子，和内容无关：“data =名称: 买水果, 金额: 20, 标签: 生活吃喝加买菜, 日期：2025-01-12，支付方式: 微信,开支类型：日常开支 ，说明: 水果购买，备注: 每天都要吃", todayDate, Msg_get),
// //                 },
// //             },
// //         },
// //     },
		
// // }
// // 构造请求数据
// 	prompt := fmt.Sprintf(`
// 		今天是 %s，请根据以下收入记录生成 JSON 数据：
// 		%s
// 		如果没有指定日期，默认使用今天；如果没有金额，请估算一个合理的数值；支付方式只有 支付宝 或微信 或银行卡 ；标签从以下内容选 生活吃喝加买菜 房贷-银行金 医疗保健 水电物业 出行 家人-互动生活穿衣用品 家用设备 电子设备 电话费 旅游 其他 摩托车 网购 学习课程；开支类型从下面选择：其他 日常开支 固定开支 社交娱乐开支 节假日开支 教育和自我提升开支 医疗保健开支 意外或紧急开支! 交通开支(出行) 加油 购物；
// 		返回的 JSON 格式如下：
// 		[
// 			{
// 				"名称": "买水果",
// 				"金额": 20,
// 				"标签": "生活吃喝加买菜",
// 				"日期": 2025-01-12,
// 				"支付方式": "微信",
//     				"开支类型": "日常开支",
// 				"备注": "记得吃水果"
// 			}
// 		]
// 		支持一次性处理多条支出记录，请确保返回的数据是 JSON 格式，不要包含无关内容或注释。
// 	`, todayDate, Msg_get)

// 	data := map[string]interface{}{
// 		"contents": []map[string]interface{}{
// 			{
// 				"parts": []map[string]interface{}{
// 					{
// 						"text": prompt,
// 					},
// 				},
// 			},
// 		},
// 	}
//     log.Println("发送的请求 data:", data)
//     payload, err := json.Marshal(data)
//     if err != nil {
//         return nil, fmt.Errorf("error marshalling data: %v", err)
//     }

//     resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
//     if err != nil {
//         log.Println("Gemini POST 请求 resp:", resp)
//         return nil, fmt.Errorf("error sending request: %v", err)
//     }
//     defer resp.Body.Close()

//     body, err := ioutil.ReadAll(resp.Body)
//     if err != nil {
//         return nil, fmt.Errorf("error reading response: %v", err)
//     }

//     if resp.StatusCode != http.StatusOK {
//         return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
//     }

//     log.Println("Gemini API response body:", string(body))

//     var apiResponse struct {
//         Candidates []struct {
//             Content struct {
//                 Parts []struct {
//                     Text string `json:"text"`
//                 } `json:"parts"`
//             } `json:"content"`
//         } `json:"candidates"`
//     }

//     if err := json.Unmarshal(body, &apiResponse); err != nil {
//         return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
//     }

//     jsonText := apiResponse.Candidates[0].Content.Parts[0].Text
//     jsonText = strings.TrimSpace(jsonText)
//     jsonText = strings.TrimPrefix(jsonText, "```json")
//     jsonText = strings.TrimSuffix(jsonText, "```")

//     log.Println("Cleaned JSON text:", jsonText)

//     var expenses []map[string]interface{}
//     if err := json.Unmarshal([]byte(jsonText), &expenses); err != nil {
//         return nil, fmt.Errorf("error unmarshalling JSON content: %v", err)
//     }

//     return expenses, nil
// }


func processRequest(Msg_get string) ([]map[string]interface{}, error) {
	log.Println("Received request with message:", Msg_get)

	todayDate := time.Now().Format("2006-01-02")
	apiKey := GetGeminiKey()
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is empty")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

	prompt := fmt.Sprintf(`
        今天是 %s，请根据以下收入记录生成 JSON 数据：
        %s
        如果没有指定日期，默认使用今天；如果没有金额，请估算一个合理的数值；支付方式只有 支付宝 或微信 或银行卡 ；标签从以下内容选 生活吃喝加买菜 房贷-银行金 医疗保健 水电物业 出行 家人-互动生活穿衣用品 家用设备 电子设备 电话费 旅游 其他 摩托车 网购 学习课程；开支类型从下面选择：其他 日常开支 固定开支 社交娱乐开支 节假日开支 教育和自我提升开支 医疗保健开支 意外或紧急开支! 交通开支(出行) 加油 购物；
        返回的 JSON 格式如下：
        [
            {
                "名称": "买水果",
                "金额": 20,
                "标签": "生活吃喝加买菜",
                "日期": "2025-01-12",
                "支付方式": "微信",
                "开支类型": "日常开支",
                "备注": "记得吃水果"
            }
        ]
        支持一次性处理多条支出记录，确保返回的数据是 JSON 格式，不要包含无关内容或注释。
    `, todayDate, Msg_get)

	requestData := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	payload, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request data: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	log.Println("Gemini API response body:", string(body)) // 记录 API 返回数据，方便调试

	var apiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling API response: %v", err)
	}

	if len(apiResponse.Candidates) == 0 || len(apiResponse.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no valid content in API response")
	}

	jsonText := strings.TrimSpace(apiResponse.Candidates[0].Content.Parts[0].Text)

	// 处理 Markdown 代码块格式
	jsonText = strings.TrimPrefix(jsonText, "```json")
	jsonText = strings.TrimPrefix(jsonText, "```")
	jsonText = strings.TrimSuffix(jsonText, "```")
	jsonText = strings.TrimSpace(jsonText)

	log.Println("Cleaned JSON text:", jsonText)

	var expenses []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonText), &expenses); err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	log.Println("expenses:", expenses)

	// 校验 JSON 数据是否符合预期格式
	for _, expense := range expenses {
	    // 检查必填字段
	    requiredFields := []string{"名称", "金额", "标签", "日期", "支付方式", "开支类型"}
	    for _, field := range requiredFields {
	        value, exists := expense[field]
	        if !exists || value == nil {
	            return nil, fmt.Errorf("missing or nil field: %s", field)
	        }
	
	        // 检查字段类型是否符合预期
	        switch field {
	        case "金额":
	            if amount, ok := value.(float64); !ok || amount <= 0 {
	                return nil, fmt.Errorf("invalid amount: %v", value)
	            }
	        case "支付方式":
	            validPaymentMethods := []string{"支付宝", "微信", "银行卡"}
	            if paymentMethod, ok := value.(string); !ok || !contains(validPaymentMethods, paymentMethod) {
	                return nil, fmt.Errorf("invalid payment method: %v", value)
	            }
	        case "日期":
	            if _, ok := value.(string); !ok {
	                return nil, fmt.Errorf("invalid date format: %v", value)
	            }
	        }
	    }
	
	    // 处理可选字段（如备注）
	    if remark, exists := expense["备注"]; exists && remark != nil {
	        if _, ok := remark.(string); !ok {
	            return nil, fmt.Errorf("invalid remark field: %v", remark)
	        }
	    } else {
	        expense["备注"] = "" // 如果备注字段不存在或为 nil，设置为空字符串
	    }
	}
	return expenses, nil
}

// 判断支付方式是否有效
func contains(validMethods []string, method string) bool {
    for _, valid := range validMethods {
        if method == valid {
            return true
        }
    }
    return false
}

// 检查日期格式是否合法 (YYYY-MM-DD)
func isValidDate(dateStr string) bool {
    _, err := time.Parse("2006-01-02", dateStr)
    return err == nil
}
func insertToNotion(databaseId, notionApiKey string, expenses []map[string]interface{}) []string {
    log.Println("expenses:", expenses)

    // 设置请求头
    headers := map[string]string{
        "Authorization":  fmt.Sprintf("Bearer %s", notionApiKey),
        "Content-Type":   "application/json",
        "Notion-Version": NOTION_API_VERSION,
    }

    var feedback []string

    // 确保 expenses 是一个 []map[string]interface{}
    // 如果 expenses 是嵌套的数组，先将其展平
    var flatExpenses []map[string]interface{}
    for _, expense := range expenses {
        if nestedExpenses, ok := expense["data"].([]map[string]interface{}); ok {
            // 如果 expense 包含嵌套的 "data" 字段
            flatExpenses = append(flatExpenses, nestedExpenses...)
        } else {
            // 否则直接添加到 flatExpenses
            flatExpenses = append(flatExpenses, expense)
        }
    }

    // 向 Notion 插入每条记录
    for _, entry := range flatExpenses {
        // 检查必填字段是否存在
        requiredFields := []string{"名称", "金额", "标签", "日期", "支付方式", "开支类型"}
        for _, field := range requiredFields {
            if _, exists := entry[field]; !exists {
                feedback = append(feedback, fmt.Sprintf("Missing required field: %s in entry: %v", field, entry))
                continue
            }
        }

        // 处理可选字段（如说明和备注）
        description := ""
        if value, exists := entry["说明"]; exists && value != nil {
            if strValue, ok := value.(string); ok {
                description = strValue
            } else {
                feedback = append(feedback, fmt.Sprintf("Invalid 说明 field in entry: %v", entry))
                continue
            }
        }

        remark := ""
        if value, exists := entry["备注"]; exists && value != nil {
            if strValue, ok := value.(string); ok {
                remark = strValue
            } else {
                feedback = append(feedback, fmt.Sprintf("Invalid 备注 field in entry: %v", entry))
                continue
            }
        }

        payload := map[string]interface{}{
            "parent": map[string]interface{}{
                "database_id": databaseId,
            },
            "properties": map[string]interface{}{
                "名称": map[string]interface{}{
                    "title": []map[string]interface{}{
                        {
                            "text": map[string]interface{}{
                                "content": entry["名称"].(string), // 确保字段名称和数据类型正确
                            },
                        },
                    },
                },
                "金额": map[string]interface{}{
                    "number": entry["金额"].(float64), // 确保字段名称和数据类型正确
                },
                "标签": map[string]interface{}{
                    "select": map[string]interface{}{
                        "name": entry["标签"].(string), // 确保字段名称和数据类型正确
                    },
                },
                "日期": map[string]interface{}{
                    "date": map[string]interface{}{
                        "start": entry["日期"].(string), // 确保字段名称和数据类型正确
                    },
                },
                "支付方式": map[string]interface{}{
                    "select": map[string]interface{}{
                        "name": entry["支付方式"].(string), // 确保字段名称和数据类型正确
                    },
                },
                "开支类型": map[string]interface{}{
                    "select": map[string]interface{}{
                        "name": entry["开支类型"].(string), // 确保字段名称和数据类型正确
                    },
                },
                "说明": map[string]interface{}{
                    "rich_text": []map[string]interface{}{
                        {
                            "text": map[string]interface{}{
                                "content": description, // 使用处理后的说明字段
                            },
                        },
                    },
                },
                "备注": map[string]interface{}{
                    "rich_text": []map[string]interface{}{
                        {
                            "text": map[string]interface{}{
                                "content": remark, // 使用处理后的备注字段
                            },
                        },
                    },
                },
            },
        }

        // 打印请求的 JSON 数据（用于调试）
        payloadBytes, _ := json.Marshal(payload)
        log.Println("Notion API request payload:", string(payloadBytes))

        // 发送请求插入数据
        req, err := http.NewRequest("POST", "https://api.notion.com/v1/pages", bytes.NewBuffer(payloadBytes))
        if err != nil {
            feedback = append(feedback, fmt.Sprintf("Error creating request for %v: %v", entry["名称"], err))
            continue
        }

        for key, value := range headers {
            req.Header.Add(key, value)
        }

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            feedback = append(feedback, fmt.Sprintf("Error inserting %v: %v", entry["名称"], err))
            continue
        }
        defer resp.Body.Close()

        // 打印 Notion API 的响应（用于调试）
        body, _ := ioutil.ReadAll(resp.Body)
        log.Println("Notion API response:", string(body))

        if resp.StatusCode == http.StatusOK {
            feedback = append(feedback, fmt.Sprintf("Successfully added: %v", entry["名称"]))
        } else {
            feedback = append(feedback, fmt.Sprintf("Failed to add %v: %v", entry["名称"], resp.Status))
        }
    }

    return feedback
}

// func insertToNotion(databaseId, notionApiKey string, expenses []map[string]interface{}) []string {
	
// 	log.Println("expenses:", expenses)


// 	// 设置请求头
// 	headers := map[string]string{
// 		"Authorization":  fmt.Sprintf("Bearer %s", notionApiKey),
// 		"Content-Type":   "application/json",
// 		"Notion-Version": NOTION_API_VERSION,
// 	}

// 	var feedback []string

// 	// 确保 expenses 是一个 []map[string]interface{}
// 	// 如果 expenses 是嵌套的数组，先将其展平
// 	var flatExpenses []map[string]interface{}
// 	for _, expense := range expenses {
// 		if nestedExpenses, ok := expense["data"].([]map[string]interface{}); ok {
// 			// 如果 expense 包含嵌套的 "data" 字段
// 			flatExpenses = append(flatExpenses, nestedExpenses...)
// 		} else {
// 			// 否则直接添加到 flatExpenses
// 			flatExpenses = append(flatExpenses, expense)
// 		}
// 	}

// 	// 向 Notion 插入每条记录
// 	for _, entry := range flatExpenses {
// 		payload := map[string]interface{}{
// 			"parent": map[string]interface{}{
// 				"database_id": databaseId,
// 			},
// 			"properties": map[string]interface{}{
// 				"名称": map[string]interface{}{
// 					"title": []map[string]interface{}{
// 						{
// 							"text": map[string]interface{}{
// 								"content": entry["名称"].(string), // 确保字段名称和数据类型正确
// 							},
// 						},
// 					},
// 				},
// 				"金额": map[string]interface{}{
// 					"number": entry["金额"].(float64), // 确保字段名称和数据类型正确
// 				},
// 				"标签": map[string]interface{}{
// 					"select": map[string]interface{}{
// 						"name": entry["标签"].(string), // 确保字段名称和数据类型正确
// 					},
// 				},
// 				"日期": map[string]interface{}{
// 					"date": map[string]interface{}{
// 						"start": entry["日期"].(string), // 确保字段名称和数据类型正确
// 					},
// 				},
// 				"支付方式": map[string]interface{}{
// 					"select": map[string]interface{}{
// 						"name": entry["支付方式"].(string), // 确保字段名称和数据类型正确
// 					},
// 				},
// 				"开支类型": map[string]interface{}{
// 					"select": map[string]interface{}{
// 						"name": entry["开支类型"].(string), // 确保字段名称和数据类型正确
// 					},
// 				},
// 				"说明": map[string]interface{}{
// 					"rich_text": []map[string]interface{}{
// 						{
// 							"text": map[string]interface{}{
// 								"content": entry["说明"].(string), // 确保字段名称和数据类型正确
// 							},
// 						},
// 					},
// 				},
// 				"备注": map[string]interface{}{
// 					"rich_text": []map[string]interface{}{
// 						{
// 							"text": map[string]interface{}{
// 								"content": entry["备注"].(string), // 确保字段名称和数据类型正确
// 							},
// 						},
// 					},
// 				},
// 			},
// 		}

// 		// 打印请求的 JSON 数据（用于调试）
// 		payloadBytes, _ := json.Marshal(payload)
// 		log.Println("Notion API request payload:", string(payloadBytes))

// 		// 发送请求插入数据
// 		req, err := http.NewRequest("POST", "https://api.notion.com/v1/pages", bytes.NewBuffer(payloadBytes))
// 		if err != nil {
// 			feedback = append(feedback, fmt.Sprintf("Error creating request for %v: %v", entry["名称"], err))
// 			continue
// 		}

// 		for key, value := range headers {
// 			req.Header.Add(key, value)
// 		}

// 		client := &http.Client{}
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			feedback = append(feedback, fmt.Sprintf("Error inserting %v: %v", entry["名称"], err))
// 			continue
// 		}
// 		defer resp.Body.Close()

// 		// 打印 Notion API 的响应（用于调试）
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		log.Println("Notion API response:", string(body))

// 		if resp.StatusCode == http.StatusOK {
// 			feedback = append(feedback, fmt.Sprintf("Successfully added: %v", entry["名称"]))
// 		} else {
// 			feedback = append(feedback, fmt.Sprintf("Failed to add %v: %v", entry["名称"], resp.Status))
// 		}
// 	}

// 	return feedback
// }
