package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"studyroom/internal/models"
	"studyroom/internal/repo"
	"studyroom/pkg/crypto"
)

type AuthService interface {
	Register(email, password string) error
	Login(email, password string) (token string, expires time.Time, err error)
	Logout(token string) error
	CurrentUser(token string) (*models.User, error)
}

type authService struct {
	users repo.UserRepo
	sess  repo.SessionRepo
}

func NewAuthService(u repo.UserRepo, s repo.SessionRepo) AuthService {
	return &authService{users: u, sess: s}
}

func (a *authService) Register(email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if !validEmail(email) || len(password) < 8 {
		return errors.New("invalid email or password")
	}
	hash, err := crypto.HashPassword(password)
	if err != nil { return err }
	_, err = a.users.Create(email, hash)
	return err
}

func (a *authService) Login(email, password string) (string, time.Time, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	uid, hash, _, err := a.users.GetByEmail(email)
	if err != nil { return "", time.Time{}, errors.New("invalid credentials") }
	if err := crypto.CheckPassword(hash, password); err != nil {
		return "", time.Time{}, errors.New("invalid credentials")
	}
	tok, _ := randomToken(32)
	exp := time.Now().Add(7*24*time.Hour).UTC()
	if err := a.sess.Create(tok, uid, exp.Format(time.RFC3339)); err != nil {
		return "", time.Time{}, err
	}
	return tok, exp, nil
}

func (a *authService) Logout(token string) error {
	if token == "" { return nil }
	return a.sess.Delete(token)
}

func (a *authService) CurrentUser(token string) (*models.User, error) {
	if token == "" { return nil, errors.New("no token") }
	uid, expStr, err := a.sess.Lookup(token)
	if err != nil { return nil, errors.New("invalid session") }
	exp, err := time.Parse(time.RFC3339, expStr)
	if err != nil || time.Now().After(exp) {
		_ = a.sess.Delete(token)
		return nil, errors.New("expired session")
	}
	email, admin, err := a.users.GetByID(uid)
	if err != nil { return nil, err }
	return &models.User{ID: uid, Email: email, IsAdmin: admin}, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil { return "", err }
	return hex.EncodeToString(b), nil
}

func validEmail(s string) bool { return strings.Contains(s, "@") && len(s) <= 255 }
