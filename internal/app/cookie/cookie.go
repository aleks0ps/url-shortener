package cookie

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	ID string `json:"id"`
	jwt.RegisteredClaims
}

const IDLength = 10
const jwtKey = "my_secret_key"

func genUniqID(length uint64) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	uniqID := make([]byte, length)
	for i := range uniqID {
		uniqID[i] = charset[r.Intn(len(charset))]
	}
	return string(uniqID)
}

func NewToken(expirationTime time.Time) (string, *Claims, error) {
	ID := genUniqID(IDLength)
	claims := &Claims{
		// generate uniq ID
		ID: ID,
		RegisteredClaims: jwt.RegisteredClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtKey))
	if err != nil {
		return "", nil, err
	}
	return tokenString, claims, nil
}

func CheckToken(tokenStr string) (*Claims, bool, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return []byte(jwtKey), nil
	})
	if err != nil {
		return &Claims{}, false, err
	}
	// expired
	if !token.Valid {
		return &Claims{}, false, nil
	}
	return claims, true, nil
}

func RefreshToken(expirationTime time.Time, tokenStr string) (string, bool, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		return "", false, err
	}
	claims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", false, err
	}
	return tokenString, true, nil
}

func NewCookie(w *http.ResponseWriter) (map[string]string, error) {
	res := make(map[string]string)
	expirationTime := time.Now().Add(5 * time.Minute)
	tokenString, claims, err := NewToken(expirationTime)
	if err != nil {
		return nil, err
	}
	http.SetCookie(*w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})
	http.SetCookie(*w, &http.Cookie{
		Name:    "id",
		Value:   claims.ID,
		Expires: expirationTime,
	})
	res["id"] = (*claims).ID
	res["token"] = tokenString
	return res, nil
}

func GetCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func ValidateCookie(r *http.Request) error {
	// plain text
	ID, err := GetCookie(r, "id")
	if err != nil {
		return err
	}
	// encoded
	tokenString, err := GetCookie(r, "token")
	if err != nil {
		return err
	}
	// decode token string
	claims, ok, err := CheckToken(tokenString)
	if err != nil {
		return err
	}
	if !ok {
		err := errors.New("token expired")
		return err
	}
	// compare
	if claims.ID != ID {
		err := errors.New("token invalid")
		return err
	}
	return nil
}

func EnsureCookie(w *http.ResponseWriter, r *http.Request, name string) (string, error) {
	cookieValue, err := GetCookie(r, name)
	if err != nil {
		// No cookie found
		// Generate new
		myCookies, err := NewCookie(w)
		if err != nil {
			return "", err
		}
		cookieValue = myCookies[name]
	}
	// return cookie value
	return cookieValue, nil
}
