package mongo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	logger "github.com/AuroScoz/HeroFishingServer/herofishingGoModule/logger"

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
		log.Errorf("%s 傳入toekn為空", logger.LOG_Mongo)
		return "", fmt.Errorf("傳入toekn為空")
	}
	log.Infof("%s APIPublicKey: %s", logger.LOG_Mongo, APIPublicKey)
	log.Infof("%s APIPrivateKey: %s", logger.LOG_Mongo, APIPrivateKey)
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
	authBodyBytes, _ := io.ReadAll(authResp.Body)
	// 取得admin access_token失敗
	if authResp.StatusCode != 200 {
		return "", fmt.Errorf("get admin access_token failed: %v, Response: %s", authResp.Status, authBodyBytes)
	}
	// 取得admin access_token成功
	var auth Response_Auth
	json.Unmarshal(authBodyBytes, &auth)

	log.Infof("%s player token: %s", logger.LOG_Mongo, token)
	log.Infof("%s admin access_token: %s", logger.LOG_Mongo, auth.AccessToken)

	// 驗證玩家token
	verifyEndpoint := fmt.Sprintf(`https://realm.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/users/verify_token`, EnvGroupID[Env], EnvAppObjID[Env])

	verifyBody := map[string]string{
		"token": token,
	}
	verifyBytes, _ := json.Marshal(verifyBody)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", verifyEndpoint, bytes.NewBuffer(verifyBytes))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	verifyResp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer verifyResp.Body.Close()
	verifyBodyBytes, _ := io.ReadAll(verifyResp.Body)
	// 驗證玩家token失敗
	if verifyResp.StatusCode != 200 {
		return "", fmt.Errorf("player token varify failed: %v, Response: %s", verifyResp.Status, verifyBodyBytes)
	}
	// 驗證玩家token成功
	var verify Response_Verify
	err = json.Unmarshal(verifyBodyBytes, &verify)
	if err != nil {
		log.Errorf("%s JSON Unmarshal error: %v", logger.LOG_Mongo, err)
		return "", err
	}
	decodedText, err := base64.StdEncoding.DecodeString(verify.Body)
	if err != nil {
		log.Errorf("%s Base64 decode error: %v", logger.LOG_Mongo, err)
		return "", err
	}
	log.Infof("%s verifyResponse.Body: %s", logger.LOG_Mongo, verify.Body)
	log.Infof("%s decodedText: %s", logger.LOG_Mongo, decodedText)
	var responseBody map[string]interface{}
	json.Unmarshal(decodedText, &responseBody)

	playerID := responseBody["custom_user_data"].(map[string]interface{})["_id"].(string)
	log.Infof("%s playerID: %s", logger.LOG_Mongo, playerID)

	return playerID, nil
}
