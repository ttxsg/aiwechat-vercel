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
)

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
		// 检查文本消息是否以 "0 " 开头
		if len(msgContent) >= 2 && msgContent[:2] == "0 " {
			Msg_get = msgContent[2:] // 去掉前面的 "0 " 进行处理
			log.Println("Msg_get:", Msg_get)
			// 进行 API 调用，替换 data_send 为 Msg_get
			expenses, err := processRequest(Msg_get)
			if err != nil {
				log.Println("Error processing request:", err)
				replyMsg = "调用processRequest失败error"
				return
			}

			// 将 expenses 转换为 JSON 字符串
			expensesJson, err := json.Marshal(expenses)
			if err != nil {
				log.Println("Error marshalling expenses to JSON:", err)
				replyMsg = "调用转换为 JSON失败error"
				return
			}
			replyMsg = string(expensesJson)

			// 调用 Notion API 插入数据
			feedback := insertToNotion(expenses)

			// 输出反馈信息
			var replyBuilder strings.Builder
			for _, message := range feedback {
				log.Println(message)
				replyBuilder.WriteString(message + "\n")
			}
			replyMsg = replyBuilder.String()
		} else {
			// 如果不是以 "0 " 开头，则使用正常的聊天处理
			bot := chat.GetChatBot(config.GetUserBotType(userId))
			replyMsg = bot.Chat(userId, msgContent)
		}
	} else {
		// 如果是其他类型的消息，使用媒体消息的处理逻辑
		bot := chat.GetChatBot(config.GetUserBotType(userId))
		replyMsg = bot.HandleMediaMsg(msg)
	}
	return
}

func processRequest(Msg_get string) ([]map[string]interface{}, error) {
	log.Println("Msg_get:", Msg_get)
	// 获取今天的日期
	todayDate := time.Now().Format("2006-01-02")
	fmt.Println("Today's date:", todayDate) // 使用 todayDate 避免未使用变量警告

	// 设置 API 请求 URL 和数据
	apiKey := GetGeminiKey()
	log.Println("apiKey:", apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is empty")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

	// 请求的数据
	data := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": fmt.Sprintf("%s 记账  %s ，如果没有金额，帮我虚拟估算一个数，支付方式只有 支付宝 或微信 或银行卡 ，标签从以下内容选 生活吃喝加买菜 房贷-银行金 医疗保健 水电物业 出行 家人-互动生活穿衣用品 家用设备 电子设备 电话费 旅游 其他 摩托车 网购 学习课程。开支类型从下面选择：其他 日常开支 固定开支 社交娱乐开支 节假日开支 教育和自我提升开支 医疗保健开支 意外或紧急开支!! 交通开支(出行) 加油 购物。时间默认是今天 你给我返回一个json的格式下面格式的内容，如果是多条json组合成的列表格式返回给我，不要和内容无关的东西，其中不需要换行符，只要：“data =名称: 买水果, 金额: 20, 标签: 生活吃喝加买菜, 日期：2025-01-12，支付方式: 微信,开支类型：日常开支 ，说明: 水果购买",",", Msg_get),
					},
				},
			},
		},
	}

	// 将数据转化为 JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling data: %v", err)
	}

	// 发送 POST 请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Gemini POST 请求 resp:", resp)
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// 检查请求是否成功
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	// 打印 Gemini API 返回的原始数据（用于调试）
	log.Println("Gemini API response body:", string(body))

	// 解析 JSON 响应
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

	// 提取文本内容并清理多余的字符
	jsonText := apiResponse.Candidates[0].Content.Parts[0].Text
	jsonText = strings.TrimSpace(jsonText) // 去除前后空格
	jsonText = strings.TrimPrefix(jsonText, "```json") // 去除开头的 ```json
	jsonText = strings.TrimSuffix(jsonText, "```") // 去除结尾的 ```

	// 打印清理后的 JSON 文本（用于调试）
	log.Println("Cleaned JSON text:", jsonText)

	// 解析 JSON 内容
	var expenses []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonText), &expenses); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON content: %v", err)
	}

	return expenses, nil
}

func insertToNotion(expenses []map[string]interface{}) []string {
	// 设置 Notion API 密钥和数据库ID
	NOTION_API_KEY := "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" // 使用 := 声明并赋值
	DATABASE_ID := "1a161e88039681848fd5e7712ee2d7d8"                   // 使用 := 声明并赋值

	if NOTION_API_KEY == "" || DATABASE_ID == "" {
		return []string{"Notion API key or database ID not set"}
	}

	// 设置请求头
	headers := map[string]string{
		"Authorization":  fmt.Sprintf("Bearer %s", NOTION_API_KEY),
		"Content-Type":   "application/json",
		"Notion-Version": "2022-06-28",
	}

	var feedback []string

	// 向 Notion 插入每条记录
	for _, entry := range expenses {
		payload := map[string]interface{}{
			"parent": map[string]interface{}{
				"database_id": DATABASE_ID,
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
								"content": entry["说明"].(string), // 确保字段名称和数据类型正确
							},
						},
					},
				},
				"备注": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{
							"text": map[string]interface{}{
								"content":entry["备注"], // 确保字段名称和数据类型正确
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
