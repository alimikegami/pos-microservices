package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

func CreateJWTToken(userID int64, userName string, externalID string, jwtSecretKey string) (string, error) {
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["userID"] = userID
	claims["name"] = userName
	claims["externalID"] = externalID
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecretKey))
}

func ExtractTokenUser(c echo.Context) (uint64, string, string) {
	user := c.Get("user").(*jwt.Token)
	if user.Valid {
		claims := user.Claims.(jwt.MapClaims)
		userID := claims["userID"].(float64)
		name := claims["name"].(string)
		externalID := claims["externalID"].(string)
		return uint64(userID), name, externalID
	}
	return 0, "", ""
}
