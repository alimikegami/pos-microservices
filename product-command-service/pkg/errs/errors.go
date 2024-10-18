package errs

import (
	"errors"
	"net/http"
)

const (
	ErrStatusInternalServer         = http.StatusInternalServerError
	ErrStatusClient                 = http.StatusBadRequest
	ErrStatusNotLoggedIn            = http.StatusUnauthorized
	ErrStatusNoPermission           = http.StatusForbidden
	ErrStatusUnauthorized           = http.StatusUnauthorized
	ErrStatusNotFound               = http.StatusNotFound
	ErrStatusEmailAlreadyUsed       = http.StatusBadRequest
	ErrStatusFileSizeExceedingLimit = http.StatusRequestEntityTooLarge
	ErrStatusConflict               = http.StatusConflict
	ErrBadGateway                   = http.StatusBadGateway
	ErrStatusPropertyBlockIsSold    = http.StatusGone
)

var (
	ErrInternalServer              = errors.New("Internal server error")
	ErrClient                      = errors.New("Bad request")
	ErrNotLoggedIn                 = errors.New("Unauthorized access")
	ErrInvalidCredentialsEmail     = errors.New("Email or password is incorrect")
	ErrUnauthorized                = errors.New("Forbidden access")
	ErrNotFound                    = errors.New("Resource not found")
	ErrAccountNotFound             = errors.New("Account not found")
	ErrEmailAlreadyUsed            = errors.New("Email has already been used")
	ErrNIKAlreadyUsed              = errors.New("NIK has already been used")
	ErrWrongPassword               = errors.New("Password is incorrect")
	ErrTokenExpired                = errors.New("The token is already expired")
	ErrNotAnImage                  = errors.New("Uploaded file is not an image")
	ErrConflict                    = errors.New("Conflicting record found")
	ErrInvalidPasswordConfirmation = errors.New("Incorrect password confirmation")
	ErrUserAlreadyExists           = errors.New("User already exists")
	ErrExpiredToken                = errors.New("Token has expired")
	ErrUnverifiedUser              = errors.New("The user is not verified yet")
	ErrPaymentExpired              = errors.New("Payment for this order has expired")
	ErrPropertyBlockIsSold         = errors.New("property block has been sold")
	ErrDuplicateName               = errors.New("Duplicate name found")
)

var errorMap = map[error]int{
	ErrInternalServer:              ErrStatusInternalServer,
	ErrInvalidCredentialsEmail:     ErrStatusUnauthorized,
	ErrUnauthorized:                ErrStatusUnauthorized,
	ErrClient:                      ErrStatusClient,
	ErrNotFound:                    ErrStatusNotFound,
	ErrEmailAlreadyUsed:            ErrStatusEmailAlreadyUsed,
	ErrUserAlreadyExists:           ErrStatusConflict,
	ErrWrongPassword:               ErrStatusUnauthorized,
	ErrTokenExpired:                ErrStatusNoPermission,
	ErrNotAnImage:                  ErrStatusClient,
	ErrConflict:                    ErrStatusConflict,
	ErrNIKAlreadyUsed:              ErrStatusEmailAlreadyUsed,
	ErrAccountNotFound:             ErrStatusNotFound,
	ErrInvalidPasswordConfirmation: ErrStatusUnauthorized,
	ErrExpiredToken:                ErrStatusUnauthorized,
	ErrUnverifiedUser:              ErrStatusNoPermission,
	ErrPaymentExpired:              ErrStatusNoPermission,
	ErrPropertyBlockIsSold:         ErrStatusPropertyBlockIsSold,
	ErrDuplicateName:               ErrStatusConflict,
}

func GetErrorStatusCode(err error) int {
	errStatusCode, ok := errorMap[err]
	if !ok {
		errStatusCode = errorMap[ErrInternalServer]
	}
	return errStatusCode
}
