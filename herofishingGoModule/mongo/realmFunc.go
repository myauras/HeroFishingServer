package mongo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type requestData struct {
	Token string `json:"Token"`
	Env   string `json:"Env"`
}

type authResponseData struct {
	AccessToken string `json:"access_token"`
}

type verifyResponseBody struct {
	CustomUserData struct {
		ID string `json:"_id"`
	} `json:"custom_user_data"`
}

type replyData struct {
	PlayerID string `json:"playerID"`
}

func PlayerTokenVerify(data requestData) (string, error) {
	const apiPublicKey = "YOUR_API_PUBLIC_KEY"   // Replace with your actual key
	const apiPrivateKey = "YOUR_API_PRIVATE_KEY" // Replace with your actual key

	// Check data format
	if data.Token == "" || data.Env == "" {
		fmt.Println("[PlayerTokenVerify] 格式錯誤")
		return "", fmt.Errorf("格式錯誤")
	}

	// Get admin access_token
	authEndpoint := "https://realm.mongodb.com/api/admin/v3.0/auth/providers/mongodb-cloud/login"
	authBody := map[string]string{
		"username": apiPublicKey,
		"apiKey":   apiPrivateKey,
	}
	authBodyBytes, _ := json.Marshal(authBody)

	resp, err := http.Post(authEndpoint, "application/json", bytes.NewBuffer(authBodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("取得admin access_token失敗")
	}

	var authResponse authResponseData
	json.Unmarshal(body, &authResponse)
	adminToken := authResponse.AccessToken

	// Verify token
	verifyEndpoint := fmt.Sprintf("https://realm.mongodb.com/api/admin/v3.0/groups/YOUR_GROUP_ID/apps/YOUR_APP_ID/users/verify_token") // Replace placeholders with actual values
	verifyBody := map[string]string{
		"token": data.Token,
	}
	verifyBodyBytes, _ := json.Marshal(verifyBody)

	req, _ := http.NewRequest("POST", verifyEndpoint, bytes.NewBuffer(verifyBodyBytes))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("玩家token驗證失敗")
	}

	var verifyBody verifyResponseBody
	json.Unmarshal(body, &verifyBody)

	// Process the response and return
	reply := replyData{
		PlayerID: verifyBody.CustomUserData.ID,
	}

	replyJSON, _ := json.Marshal(reply)
	return string(replyJSON), nil
}

func main() {
	// Sample usage
	data := requestData{
		Token: "sample_token",
		Env:   "sample_env",
	}
	result, err := PlayerTokenVerify(data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(result)
}
