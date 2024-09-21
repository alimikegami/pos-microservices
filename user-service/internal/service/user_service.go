package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alimikegami/e-commerce/user-service/config"
	"github.com/alimikegami/e-commerce/user-service/internal/domain"
	"github.com/alimikegami/e-commerce/user-service/internal/dto"
	"github.com/alimikegami/e-commerce/user-service/internal/repository"
	"github.com/alimikegami/e-commerce/user-service/pkg/errs"
	"github.com/alimikegami/e-commerce/user-service/pkg/utils"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	AddUser(ctx context.Context, data dto.UserRequest) (err error)
	Login(ctx context.Context, payload dto.UserRequest) (respPayload dto.LoginResponse, err error)
	UpdateUser(ctx context.Context, payload dto.UserRequest) (err error)
}

type ServiceImpl struct {
	repo          repository.UserRepository
	config        config.Config
	kafkaProducer *kafka.Conn
}

func CreateNewService(repo repository.UserRepository, config config.Config, kafkaProducer *kafka.Conn) UserService {
	return &ServiceImpl{repo: repo, config: config, kafkaProducer: kafkaProducer}
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

	token, err := utils.CreateJWTToken(user.ID, user.Name, user.ExternalID, s.config.JWTSecret)
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
	}

	if err := s.repo.UpdateUser(ctx, updatedUserData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	kafkaMsg := dto.KafkaMessage{
		EventType: "user_update",
		Data: dto.UserResponse{
			ExternalID: userData.ExternalID,
			Name:       payload.Name,
		},
	}

	jsonMsg, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	// Implement retry logic for Kafka producer
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.writeKafkaMessage(jsonMsg)
		if err == nil {
			break
		}
		log.Printf("Failed to write Kafka message (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
	}

	if err != nil {
		return fmt.Errorf("failed to write Kafka message after %d attempts: %w", maxRetries, err)
	}

	return nil
}

func (s *ServiceImpl) writeKafkaMessage(msg []byte) error {
	_, err := s.kafkaProducer.WriteMessages(
		kafka.Message{
			Value: msg,
		},
	)
	return err
}
