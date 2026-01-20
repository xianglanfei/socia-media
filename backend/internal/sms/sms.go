package sms

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// VerificationCode represents a SMS verification code
type VerificationCode struct {
	Phone     string    `json:"phone"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SMSService interface for SMS providers
type SMSService interface {
	SendCode(phone string, code string) error
	VerifyCode(phone string, code string) error
}

// MockSMSService is a mock implementation for testing
type MockSMSService struct {
	verificationCodes map[string]VerificationCode
}

// NewMockSMSService creates a new mock SMS service
func NewMockSMSService() *MockSMSService {
	return &MockSMSService{
		verificationCodes: make(map[string]VerificationCode),
	}
}

// SendCode generates and stores a verification code
func (m *MockSMSService) SendCode(phone string, code string) error {
	// Generate a random 6-digit code if not provided
	if code == "" {
		code = generateRandomCode()
	}

	m.verificationCodes[phone] = VerificationCode{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5 minute expiry
	}

	fmt.Printf("Mock SMS sent to %s: %s\n", phone, code)
	return nil
}

// VerifyCode verifies a verification code
func (m *MockSMSService) VerifyCode(phone string, code string) error {
	storedCode, exists := m.verificationCodes[phone]
	if !exists {
		return fmt.Errorf("no verification code found for phone %s", phone)
	}

	if time.Now().After(storedCode.ExpiresAt) {
		delete(m.verificationCodes, phone)
		return fmt.Errorf("verification code expired")
	}

	if storedCode.Code != code {
		return fmt.Errorf("invalid verification code")
	}

	// Delete the code after successful verification
	delete(m.verificationCodes, phone)
	return nil
}

// generateRandomCode generates a random 6-digit code
func generateRandomCode() string {
	const digits = "0123456789"
	const length = 6

	b := make([]byte, length)
	for i := range b {
		b[i] = digits[rand.Intn(len(digits))]
	}
	return string(b)
}

// StoreVerificationCode stores a verification code in Redis
func StoreVerificationCode(redisClient interface{}, key string, code string, ttl time.Duration) error {
	// This would use Redis in a real implementation
	// For now, just return success
	return nil
}

// RetrieveVerificationCode retrieves a verification code from Redis
func RetrieveVerificationCode(redisClient interface{}, key string) (string, error) {
	// This would use Redis in a real implementation
	// For now, just return the code
	return "123456", nil
}

// DeleteVerificationCode deletes a verification code from Redis
func DeleteVerificationCode(redisClient interface{}, key string) error {
	// This would use Redis in a real implementation
	// For now, just return success
	return nil
}

// GenerateVerificationCodePayload generates a verification code request payload
type GenerateVerificationCodePayload struct {
	Phone string `json:"phone"`
}

// VerificationCodeResponse is the response for a verification code request
type VerificationCodeResponse struct {
	Message string `json:"message"`
	Phone   string `json:"phone"`
	Code    string `json:"code,omitempty"`
}
