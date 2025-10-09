package crypto

import "golang.org/x/crypto/bcrypt"

func HashPassword(pw string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

func CheckPassword(hash []byte, pw string) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(pw))
}
