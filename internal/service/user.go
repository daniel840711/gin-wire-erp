package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"interchange/internal/core"
	"interchange/internal/database/mongodb/model"
	"interchange/internal/database/mongodb/repository"
	"interchange/internal/dto"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/telemetry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	trace    *telemetry.Trace
	userRepo *repository.UserRepository
}

func NewUserService(trace *telemetry.Trace, userRepo *repository.UserRepository) *UserService {
	return &UserService{trace: trace, userRepo: userRepo}
}

// 新增用戶（管理專用，input/output 皆為 DTO）
func (s *UserService) CreateUser(ctx context.Context, dto *dto.CreateUserDto) (*dto.UserResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	user := &model.User{
		ID:          primitive.NewObjectID(),
		ExternalID:  dto.ExternalID,
		DisplayName: dto.DisplayName,
		Email:       dto.Email,
		Role:        dto.Role,
		Status:      dto.Status,
		CreatedAt:   time.Now().UTC(),
	}
	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, cErr.DatabaseError("database CreateUser error")
	}
	return modelToUserResponseDto(created), nil
}

// 依 id 查詢
func (s *UserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*dto.UserResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, cErr.NotFound("user not found")
		}
		return nil, cErr.DatabaseError("database GetUserByID error")
	}

	return modelToUserResponseDto(user), nil
}

// 管理後台列舉用戶（支援分頁、篩選，回傳 Response DTO）
func (s *UserService) ListUsers(ctx context.Context, filter bson.M, page, size int64) ([]*dto.UserResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)
	opts := core.ListOptions{
		Filter: filter,
		Page:   page,
		Size:   size,
	}
	users, err := s.userRepo.List(ctx, opts)
	if err != nil {
		return nil, cErr.DatabaseError("database ListUsers error")
	}
	resp := make([]*dto.UserResponseDto, len(users))
	for i, u := range users {
		resp[i] = modelToUserResponseDto(u)
	}

	return resp, nil
}

// 更新用戶基本資訊（input DTO）
func (s *UserService) UpdateUserByID(ctx context.Context, id primitive.ObjectID, dto *dto.UpdateUserDto) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	update := bson.M{}
	if dto.ExternalID != nil {
		update["externalID"] = *dto.ExternalID
	}
	if dto.DisplayName != nil {
		update["displayName"] = *dto.DisplayName
	}
	if dto.Email != nil {
		update["email"] = *dto.Email
	}
	if dto.Lang != nil {
		update["lang"] = *dto.Lang
	}
	if dto.Role != nil {
		update["role"] = *dto.Role
	}
	if dto.Status != nil {
		update["status"] = *dto.Status
	}

	matchedCount, err := s.userRepo.UpdateByID(ctx, id, update)
	if err != nil {
		return cErr.DatabaseError("database UpdateUserByID error")
	}
	if matchedCount == 0 {
		notFound := cErr.NotFound(fmt.Sprintf("user with id %s not found", id.Hex()))
		return notFound
	}
	return nil
}

// 專屬：修改用戶狀態
func (s *UserService) UpdateUserStatus(ctx context.Context, id primitive.ObjectID, dto *dto.UpdateUserStatusDto) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	matchedCount, err := s.userRepo.UpdateStatus(ctx, id, dto.Status)
	if err != nil {
		return cErr.DatabaseError("database UpdateUserStatus error")
	}
	if matchedCount == 0 {
		notFound := cErr.NotFound(fmt.Sprintf("user with id %s not found", id.Hex()))
		return notFound
	}
	return nil
}
func (s *UserService) UpdateUserLastSeen(ctx context.Context, id primitive.ObjectID) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	matchedCount, err := s.userRepo.UpdateLastSeen(ctx, id, time.Now().UTC())
	if err != nil {
		return cErr.DatabaseError("database UpdateUserLastSeen error")
	}
	if matchedCount == 0 {
		notFound := cErr.NotFound(fmt.Sprintf("user with id %s not found", id.Hex()))
		return notFound
	}
	return nil
}

// 專屬：修改用戶角色
func (s *UserService) UpdateUserRole(ctx context.Context, id primitive.ObjectID, dto *dto.UpdateUserRoleDto) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	matchedCount, err := s.userRepo.UpdateRole(ctx, id, dto.Role)
	if err != nil {
		return cErr.DatabaseError("database UpdateUserRole error")
	}
	if matchedCount == 0 {
		notFound := cErr.NotFound(fmt.Sprintf("user with id %s not found", id.Hex()))
		return notFound
	}
	return nil
}

// 刪除用戶
func (s *UserService) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userRepo.DeleteByID(ctx, id); err != nil {
		return cErr.DatabaseError("database DeleteUser error")
	}
	return nil
}

// 管理端：找出 N 天未活躍用戶
func (s *UserService) ListInactiveUsers(ctx context.Context, sinceDays int) ([]*dto.UserResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)
	since := time.Now().Add(-time.Duration(sinceDays) * 24 * time.Hour)
	filter := bson.M{"last_seen": bson.M{"$lt": since}}
	return s.ListUsers(ctx, filter, 0, 1000)
}

func modelToUserResponseDto(m *model.User) *dto.UserResponseDto {
	resp := &dto.UserResponseDto{
		ID:          m.ID.Hex(),
		ExternalID:  m.ExternalID,
		DisplayName: m.DisplayName,
		Email:       m.Email,
		Role:        m.Role,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.LastSeen == nil || !m.LastSeen.IsZero() {
		resp.LastSeen = m.LastSeen
	}
	return resp
}
