package margin_monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// LoginResponse 登录响应结构
type LoginResponse struct {
	AccessToken string `json:"access_token"` // JWT 令牌
}

// 登录函数
func login(loginUrl, username, password string) (string, error) {
	// 登录请求的URL
	url := fmt.Sprintf("%v/api/v1/token/login", loginUrl)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("")))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// 设置 Basic Auth 头
	req.SetBasicAuth(username, password)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to login, status code: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// 解析JSON响应，提取JWT令牌
	var loginResponse LoginResponse
	err = json.Unmarshal(body, &loginResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return loginResponse.AccessToken, nil
}

func RestartBot(restartAPI string, username string, password string) error {
	restartToken, err := login(restartAPI, username, password)
	if err != nil {
		log.Printf("failed to get bot restartToken: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%v/api/v1/reload_config", restartAPI), bytes.NewBuffer([]byte("")))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// 添加Authorization头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", restartToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
