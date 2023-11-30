package mongo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	logger "herofishingGoModule/logger"

	log "github.com/sirupsen/logrus"
)

// 取admin access_token的回傳參數
type Response_Auth struct {
	AccessToken string `json:"access_token"`
}

// 檢查帳戶驗證結果的回傳參數
// 回傳格式大概長這樣
// {"sub":"64f8a5ab9f2aeb56e0c02c4e","exp":1696434413,"iat":1696432613,"iss":"64f8a5c09f2aeb56e0c02c9a","custom_user_data":{"_id":"64f8a5ab9f2aeb56e0c02c4e","createdAt":{"$date":{"$numberLong":"1694016939834"}},"role":"Player"},"domain_id":"64e6d784c96a30ebafdf3de2","device_id":"64f8a5bf9f2aeb56e0c02c99"}
type Response_Verify struct {
	CustomUserData map[string]interface{} `json:"custom_user_data"`
}

// 驗證玩家帳戶，成功時返回playerID
func PlayerVerify(token string) (string, error) {
	log.Infof("%s PlayerVerify: %s", logger.LOG_Mongo, token)
	if token == "" {
		log.Errorf("%s 傳入toekn為空", logger.LOG_Mongo)
		return "", fmt.Errorf("傳入toekn為空")
	}
	// log.Infof("%s APIPublicKey: %s", logger.LOG_Mongo, APIPublicKey)
	// log.Infof("%s APIPrivateKey: %s", logger.LOG_Mongo, APIPrivateKey)
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

	// log.Infof("%s player token: %s", logger.LOG_Mongo, token)
	// log.Infof("%s admin access_token: %s", logger.LOG_Mongo, auth.AccessToken)

	// 驗證玩家token
	verifyEndpoint := fmt.Sprintf(`https://realm.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/users/verify_token`, EnvGroupID[Env], EnvAppObjID[Env])
	log.Infof("%s verifyEndpoint: %s", logger.LOG_Mongo, verifyEndpoint)
	log.Infof("%s token: %s", logger.LOG_Mongo, token)

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
	log.Infof("%s verifyBodyBytes: %s", logger.LOG_Mongo, verifyBodyBytes)
	err = json.Unmarshal(verifyBodyBytes, &verify)
	if err != nil {
		log.Errorf("%s JSON Unmarshal error: %v", logger.LOG_Mongo, err)
		return "", err
	}
	if verify.CustomUserData == nil {
		log.Errorf("%s CustomUserData is nil", logger.LOG_Mongo)
		return "", fmt.Errorf("CustomUserData is nil")
	}

	playerID, ok := verify.CustomUserData["_id"].(string)
	if !ok {
		log.Errorf("%s Failed to extract playerID from CustomUserData", logger.LOG_Mongo)
		return "", fmt.Errorf("Failed to extract playerID")
	}

	log.Infof("%s PlayerVerify成功 playerID: %s", logger.LOG_Mongo, playerID)

	return playerID, nil
}
