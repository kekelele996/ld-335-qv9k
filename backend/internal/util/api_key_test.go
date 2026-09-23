package util

import "testing"

func TestAPIKeyHashAndVerify(t *testing.T) {
	secret := "api-secret-xyz"
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	hash := HashAPIKey(key, secret)
	tests := []struct {
		name    string
		key     string
		secret  string
		wantOK  bool
	}{
		{name: "correct key", key: key, secret: secret, wantOK: true},
		{name: "wrong key", key: "ak_wrong", secret: secret, wantOK: false},
		{name: "wrong secret", key: key, secret: "other-secret", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyAPIKey(tt.key, hash, tt.secret); got != tt.wantOK {
				t.Fatalf("VerifyAPIKey() = %v, want %v", got, tt.wantOK)
			}
		})
	}
}
