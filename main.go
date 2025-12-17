package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"gopkg.in/gomail.v2"
)

// 请求和响应结构体
type ForgotPasswordRequest struct {
	Email    string `json:"email"`
	ResetURL string `json:"reset_url"`
}

type ForgotPasswordResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	ResetToken string `json:"reset_token,omitempty"`
}

// 模拟数据库存储
var resetTokens = make(map[string]time.Time)

// 获取本机局域网IP
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("无法获取局域网IP")
}

// 发送重置密码邮件
func sendResetPasswordEmail(toEmail, token, resetURL string) error {
	// 构建完整链接
	resetLink := fmt.Sprintf("%s?token=%s", resetURL, url.QueryEscape(token))

	// 邮件内容
	message := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>密码重置请求</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px;">
        <div style="background-color: #4CAF50; color: white; padding: 10px; text-align: center; border-radius: 5px 5px 0 0;">
            <h2>密码重置请求</h2>
        </div>
        <div style="padding: 20px;">
            <p>尊敬的用户，您好！</p>
            <p>我们收到了您重置密码的请求。请点击以下按钮重置您的密码：</p>
            
            <p style="text-align: center;">
                <a href="%s" style="display: inline-block; padding: 12px 24px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px; margin: 10px 0;">
                    重置密码
                </a>
            </p>
            
            <p>如果按钮无法点击，请复制以下链接到浏览器地址栏中打开：</p>
            <div style="background-color: #f5f5f5; padding: 10px; border-radius: 3px; word-break: break-all; font-family: monospace; font-size: 12px;">
                %s
            </div>
            
            <p><strong>重要提示：</strong></p>
            <ul>
                <li>此链接仅在30分钟内有效</li>
                <li>如果您没有请求重置密码，请忽略此邮件</li>
                <li>请不要将此链接分享给他人</li>
            </ul>
            
            <p>如果您有任何问题，请联系我们的客服。</p>
            <p>谢谢！</p>
        </div>
        <div style="margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 12px;">
            <p>此邮件由系统自动发送，请勿直接回复。</p>
            <p>© %d 公司名称 版权所有</p>
        </div>
    </div>
</body>
</html>
`, resetLink, resetLink, time.Now().Year())

	host := "smtp.qq.com"
	port := 25
	userName := "1806646639@qq.com"
	password := "pqbrphwuxfihiajf"

	m := gomail.NewMessage()
	m.SetHeader("From", userName)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "密码重置请求 - 您的账户")
	m.SetBody("text/html", message)

	d := gomail.NewDialer(
		host,
		port,
		userName,
		password,
	)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return d.DialAndSend(m)
}

// 忘记密码处理函数
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头部
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只接受POST请求
	if r.Method != "POST" {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ForgotPasswordResponse{
			Success: false,
			Message: "无效的请求数据",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证邮箱
	if req.Email == "" {
		response := ForgotPasswordResponse{
			Success: false,
			Message: "邮箱不能为空",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查重置URL
	if req.ResetURL == "" {
		response := ForgotPasswordResponse{
			Success: false,
			Message: "重置URL不能为空",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 生成重置令牌
	resetToken := uuid.New().String()

	// 存储令牌（30分钟有效）
	resetTokens[resetToken] = time.Now().Add(30 * time.Minute)

	// 构建完整的重置链接
	resetLink := fmt.Sprintf("%s?token=%s", req.ResetURL, resetToken)

	log.Printf("发送重置邮件到: %s", req.Email)
	log.Printf("重置链接: %s", resetLink)

	// 发送邮件
	err := sendResetPasswordEmail(req.Email, resetToken, req.ResetURL)
	if err != nil {
		log.Printf("发送邮件失败: %v", err)
		response := ForgotPasswordResponse{
			Success: false,
			Message: "邮件发送失败，请稍后重试",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回成功响应
	response := ForgotPasswordResponse{
		Success:    true,
		Message:    "重置链接已发送到您的邮箱",
		ResetToken: resetToken,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// 重置密码验证函数
func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "GET" {
		// 获取token参数
		token := r.URL.Query().Get("token")

		if token == "" {
			response := map[string]interface{}{
				"valid":   false,
				"message": "无效的token",
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// 检查token是否存在且未过期
		expiry, exists := resetTokens[token]
		if !exists {
			response := map[string]interface{}{
				"valid":   false,
				"message": "token不存在或已过期",
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if time.Now().After(expiry) {
			response := map[string]interface{}{
				"valid":   false,
				"message": "token已过期",
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// token有效
		response := map[string]interface{}{
			"valid":   true,
			"message": "token验证成功",
		}
		json.NewEncoder(w).Encode(response)
	}
}

func main() {
	// 获取并显示本机IP
	ip, err := getLocalIP()
	if err != nil {
		log.Printf("无法获取局域网IP: %v", err)
		ip = "127.0.0.1"
	}

	log.Printf("服务器IP地址: %s", ip)
	log.Printf("建议前端使用: http://%s:3000", ip)

	// 注册API路由
	http.HandleFunc("/api/forgot-password", forgotPasswordHandler)
	http.HandleFunc("/api/validate-reset-token", resetPasswordHandler)

	// 启动服务器
	port := ":8080"
	log.Printf("Go后端服务器启动在 http://localhost%s", port)
	log.Println("API端点: POST http://localhost:8080/api/forgot-password")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
