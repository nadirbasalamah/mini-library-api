package rents

import (
	"context"
	"go-digital-library-fiber/pkg/utils"

	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, pagination utils.Pagination) ([]Rent, error)
	GetByUser(ctx context.Context, userId uint) ([]Rent, error)
	GetByID(ctx context.Context, id uint) (Rent, error)
	Create(ctx context.Context, rentReq *RentRequest) error
	Update(ctx context.Context, rentReq *UpdateRentRequest, id uint) (Rent, error)
	Delete(ctx context.Context, id uint) error
}

type service struct {
	getall
	getbyuser
	get
	create
	update
	delete
}

var _ Service = (*service)(nil)

func New(repository *gorm.DB) Service {
	return service{
		getall:    getall{repository: repository},
		getbyuser: getbyuser{repository: repository},
		get:       get{repository: repository},
		create:    create{repository: repository},
		update:    update{repository: repository, get: get{repository: repository}},
		delete:    delete{repository: repository, get: get{repository: repository}},
	}
}
