package group

import (
	"errors"
	"time"

	v1Group "github.com/KubeOperator/kubepi/internal/model/v1/group"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	"github.com/KubeOperator/kubepi/internal/service/v1/role"
	"github.com/KubeOperator/kubepi/internal/service/v1/rolebinding"
	costomStorm "github.com/KubeOperator/kubepi/pkg/storm"
	"github.com/KubeOperator/kubepi/pkg/util/lang"
	"github.com/asdine/storm/v3/q"
	"github.com/google/uuid"
)

type Service interface {
	common.DBService
	Create(u *v1Group.Group, options common.DBOptions) error
	List(options common.DBOptions) ([]v1Group.Group, error)
	GetByName(el string, options common.DBOptions) (*v1Group.Group, error)
	Delete(name string, options common.DBOptions) error
	Search(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1Group.Group, int, error)
	Update(name string, u *v1Group.Group, options common.DBOptions) error
}

func NewService() Service {
	return &service{}
}

type service struct {
	common.DefaultDBService
	rolebindingService rolebinding.Service
	roleService        role.Service
}

func (u *service) Update(name string, us *v1Group.Group, options common.DBOptions) error {
	cu, err := u.GetByName(name, options)
	if err != nil {
		return err
	}
	//if cu.BuiltIn {
	//	return errors.New("can not delete this resource,because it created by system")
	//}
	db := u.GetDB(options)
	us.UUID = cu.UUID
	us.CreateAt = cu.CreateAt
	us.UpdateAt = time.Now()
	return db.Update(us)
}

func (u *service) Search(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1Group.Group, int, error) {
	db := u.GetDB(options)

	var ms []q.Matcher
	for k := range conditions {
		if conditions[k].Field == "quick" {
			ms = append(ms, q.Or(
				costomStorm.Like("Name", conditions[k].Value),
				costomStorm.Like("NickName", conditions[k].Value),
			))
		} else {
			field := lang.FirstToUpper(conditions[k].Field)
			value := lang.ParseValueType(conditions[k].Value)

			switch conditions[k].Operator {
			case "eq":
				ms = append(ms, q.Eq(field, value))
			case "ne":
				ms = append(ms, q.Not(q.Eq(field, value)))
			case "like":
				ms = append(ms, costomStorm.Like(field, value.(string)))
			case "not like":
				ms = append(ms, q.Not(costomStorm.Like(field, value.(string))))
			}
		}
	}
	query := db.Select(ms...).OrderBy("CreateAt").Reverse()
	count, err := query.Count(&v1Group.Group{})
	if err != nil {
		return nil, 0, err
	}
	if size != 0 {
		query.Limit(size).Skip((num - 1) * size)
	}
	users := make([]v1Group.Group, 0)
	if err := query.Find(&users); err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (u *service) GetByName(el string, options common.DBOptions) (*v1Group.Group, error) {
	db := u.GetDB(options)
	var us v1Group.Group
	query := db.Select(q.Or(q.Eq("Name", el)))

	if err := query.First(&us); err != nil {
		return nil, err
	}
	return &us, nil
}

func (u *service) List(options common.DBOptions) ([]v1Group.Group, error) {
	db := u.GetDB(options)
	users := make([]v1Group.Group, 0)
	if err := db.All(&users); err != nil {
		return nil, err
	}
	return users, nil
}

func (u *service) Delete(name string, options common.DBOptions) error {
	db := u.GetDB(options)
	item, err := u.GetByName(name, options)
	if err != nil {
		return err
	}
	if item.BuiltIn {
		return errors.New("can not delete this resource,because it created by system")
	}
	return db.DeleteStruct(item)
}

func (u *service) Create(us *v1Group.Group, options common.DBOptions) error {
	db := u.GetDB(options)
	us.UUID = uuid.New().String()
	us.CreateAt = time.Now()
	us.UpdateAt = time.Now()
	return db.Save(us)
}
