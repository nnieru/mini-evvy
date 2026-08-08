package dto

import "github.com/nnieru/mini-evvy/internal/model"

type AuthResponseDTO struct {
	User  UserResponseDTO `json:"user"`
	Token string          `json:"token"`
}

func NewAuthResponseDTO(user *model.User, token string) AuthResponseDTO {
	return AuthResponseDTO{
		User:  NewUserResponseDTO(user),
		Token: token,
	}
}
