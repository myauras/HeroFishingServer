package mongo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	logger "herofishingGoModule/logger"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// 取admin access_token的回傳參數
type Response_Auth struct {
	AccessToken string `json:"access_token"`
}

// 檢查帳戶驗證結果的回傳參數
type Response_Verify struct {
	Body       string `json:"body"`
	StatusCode int    `json:"statusCode"`
}

// 驗證玩家帳戶，成功時返回playerID
func PlayerVerify(token string) (string, error) {
	if token == "" {
		log.Infof("%s 傳入toekn為空", logger.LOG_Mongo)
		return "", fmt.Errorf("傳入toekn為空")
	}

	// 使用 MongoDB Realm Admin API 可以參考官方文件: https://www.mongodb.com/docs/atlas/app-services/admin/api/v3/#section/Project-and-Application-IDs
	// 取得admin access_token
	authEndpoint := "https://realm.mongodb.com/api/admin/v3.0/auth/providers/mongodb-cloud/login"
	authBody := map[string]string{
		"username": APIPublicKey,
		"apiKey":   APIPrivateKey,
	}
	authBytes, _ := json.Marshal(authBody)
	authResp, err := http.Post(authEndpoint, "application/json", bytes.NewBuffer(authBytes))
	if err != nil {
		return "", err
	}
	defer authResp.Body.Close()

	// 取得admin access_token失敗
	if authResp.StatusCode != 200 {
		return "", fmt.Errorf("取得admin access_token失敗")
	}
	// 取得admin access_token成功
	authBodyBytes, _ := io.ReadAll(authResp.Body)
	var authResponse Response_Auth
	json.Unmarshal(authBodyBytes, &authResponse)

	// 驗證玩家token
	verifyEndpoint := fmt.Sprintf(`https://realm.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/users/verify_token`, EnvGroupID[Env], EnvAppObjID[Env])
	verifyBody := map[string]string{
		"token": token,
	}
	verifyBytes, _ := json.Marshal(verifyBody)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", verifyEndpoint, bytes.NewBuffer(verifyBytes))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
	verifyResp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer verifyResp.Body.Close()
	// 驗證玩家token失敗
	if verifyResp.StatusCode != 200 {
		return "", fmt.Errorf("玩家token驗證失敗")
	}
	// 驗證玩家token成功
	verifyBodyBytes, _ := io.ReadAll(verifyResp.Body)
	var verifyResponse Response_Verify
	json.Unmarshal(verifyBodyBytes, &verifyResponse)
	decodedText, _ := base64.StdEncoding.DecodeString(verifyResponse.Body)
	var responseBody map[string]interface{}
	json.Unmarshal(decodedText, &responseBody)

	playerID := responseBody["custom_user_data"].(map[string]interface{})["_id"].(string)
	log.Infof("%s playerID: %s", logger.LOG_Mongo, playerID)

	return playerID, nil
}
