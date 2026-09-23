package util

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateAPIKey 生成 API Key（ak_ 前缀 + 随机串），仅在创建时返回一次。
func GenerateAPIKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ak_" + hex.EncodeToString(b), nil
}

// HashAPIKey 使用 HMAC-SHA256 对 API Key 加盐哈希（api_key_hash）。
func HashAPIKey(apiKey, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(apiKey))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyAPIKey 校验 API Key 与哈希是否匹配。
func VerifyAPIKey(apiKey, hash, secret string) bool {
	expected := HashAPIKey(apiKey, secret)
	return hmac.Equal([]byte(expected), []byte(hash))
}

// BatchNo 生成批次号（B + 日期 + 序号）。
func BatchNo(seq int64) string {
	return fmt.Sprintf("B%s%06d", timeNowDate(), seq)
}

// SettlementNo 生成结算单号（S + 日期 + 序号）。
func SettlementNo(seq int64) string {
	return fmt.Sprintf("S%s%06d", timeNowDate(), seq)
}
