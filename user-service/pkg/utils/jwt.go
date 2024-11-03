package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

func CreateJWTToken(userID int64, userName string, externalID string, jwtSecretKey string, jwtKid string) (string, error) {
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["userID"] = userID
	claims["name"] = userName
	claims["externalID"] = externalID
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = jwtKid

	return token.SignedString([]byte(jwtSecretKey))
}

func ExtractTokenUser(c echo.Context) (uint64, uint64, string) {
	user := c.Get("user").(*jwt.Token)
	if user.Valid {
		claims := user.Claims.(jwt.MapClaims)
		userID := claims["userID"].(float64)
		roleID := claims["roleID"].(float64)
		externalID := claims["externalID"].(string)
		return uint64(userID), uint64(roleID), externalID
	}
	return 0, 0, ""
}
