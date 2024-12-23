package service

import (
	"context"
	"fmt"

	"github.com/alimikegami/pos-microservices/user-service/config"
	"github.com/alimikegami/pos-microservices/user-service/internal/domain"
	"github.com/alimikegami/pos-microservices/user-service/internal/dto"
	"github.com/alimikegami/pos-microservices/user-service/internal/repository"
	pkgdto "github.com/alimikegami/pos-microservices/user-service/pkg/dto"
	"github.com/alimikegami/pos-microservices/user-service/pkg/errs"
	"github.com/alimikegami/pos-microservices/user-service/pkg/utils"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	AddUser(ctx context.Context, data dto.UserRequest) (err error)
	Login(ctx context.Context, payload dto.UserRequest) (respPayload dto.LoginResponse, err error)
	UpdateUser(ctx context.Context, payload dto.UserRequest) (err error)
	GetUsers(ctx context.Context, filter pkgdto.Filter) (resp pkgdto.Pagination, err error)
}

type ServiceImpl struct {
	repo   repository.UserRepository
	config config.Config
}

func CreateNewService(repo repository.UserRepository, config config.Config) UserService {
	return &ServiceImpl{repo: repo, config: config}
}

func (s *ServiceImpl) AddUser(ctx context.Context, data dto.UserRequest) (err error) {
	user, err := s.repo.GetUserByEmail(ctx, data.Email)
	if err != nil {
		return
	}

	if user.ID != 0 {
		return errs.ErrEmailAlreadyUsed
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.MinCost)
	if err != nil {
		return err
	}

	userEnt := domain.User{
		Name:           data.Name,
		Email:          data.Email,
		HashedPassword: string(hash),
		ExternalID:     ulid.Make().String(),
		RoleID:         data.RoleID,
	}

	_, err = s.repo.AddUser(ctx, userEnt)
	if err != nil {
		return err
	}

	return nil
}

func (s *ServiceImpl) Login(ctx context.Context, payload dto.UserRequest) (respPayload dto.LoginResponse, err error) {
	user, err := s.repo.GetUserByEmail(ctx, payload.Email)
	if err != nil {
		return
	}

	if user.ID == 0 {
		return respPayload, errs.ErrAccountNotFound
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(payload.Password))
	if err != nil {
		log.Error().Err(err).Str("component", "Login").Msg("")
		return respPayload, errs.ErrInvalidCredentialsEmail
	}

	token, err := utils.CreateJWTToken(user.ID, user.Name, user.ExternalID, s.config.JWTConfig.JWTSecret, s.config.JWTConfig.JWTKid)
	if err != nil {
		return
	}

	respPayload.Token = token
	respPayload.UserID = user.ID

	return
}

func (s *ServiceImpl) UpdateUser(ctx context.Context, payload dto.UserRequest) error {
	// Wrap the entire function in a defer to recover from panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in UpdateUser: %v", r)
		}
	}()

	userData, err := s.repo.GetUserByID(ctx, payload.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	updatedUserData := domain.User{
		ID:             int64(payload.ID),
		Name:           payload.Name,
		Email:          userData.Email,
		HashedPassword: userData.HashedPassword,
		ExternalID:     userData.ExternalID,
		RoleID:         userData.RoleID,
	}

	if err := s.repo.UpdateUser(ctx, updatedUserData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (s *ServiceImpl) GetUsers(ctx context.Context, filter pkgdto.Filter) (resp pkgdto.Pagination, err error) {
	datas, err := s.repo.GetUsers(ctx, filter)
	if err != nil {
		return
	}

	userCount, err := s.repo.CountUsers(ctx, filter)
	if err != nil {
		return
	}

	var users []dto.UserResponse

	for _, data := range datas {
		users = append(users, dto.UserResponse{
			ID:         data.ID,
			ExternalID: data.ExternalID,
			Name:       data.Name,
			Email:      data.Email,
		})
	}

	resp.Records = users
	resp.Metadata.Limit = filter.Limit
	resp.Metadata.Page = uint64(filter.Page)
	resp.Metadata.TotalCount = uint64(userCount)

	return
}
