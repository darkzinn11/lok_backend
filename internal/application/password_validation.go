package application

import (
	"fmt"
	"unicode"

	"lockcenter-backend/internal/domain"
)

// ValidatePassword checks if a password meets the security requirements:
// - At least 8 characters
// - At least one letter
// - At least one number
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("%w: a senha deve ter no mínimo 8 caracteres", domain.ErrValidation)
	}

	var hasLetter, hasNumber bool
	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasLetter {
		return fmt.Errorf("%w: a senha deve conter pelo menos uma letra", domain.ErrValidation)
	}
	if !hasNumber {
		return fmt.Errorf("%w: a senha deve conter pelo menos um número", domain.ErrValidation)
	}

	return nil
}
