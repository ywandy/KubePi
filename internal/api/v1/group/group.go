package group

import (
	"errors"
	"fmt"
	"github.com/KubeOperator/kubepi/internal/service/v1/group"
	"github.com/KubeOperator/kubepi/pkg/kubernetes"

	"github.com/KubeOperator/kubepi/internal/api/v1/commons"
	"github.com/KubeOperator/kubepi/internal/api/v1/session"
	v1 "github.com/KubeOperator/kubepi/internal/model/v1"
	v1Role "github.com/KubeOperator/kubepi/internal/model/v1/role"
	"github.com/KubeOperator/kubepi/internal/server"
	"github.com/KubeOperator/kubepi/internal/service/v1/cluster"
	"github.com/KubeOperator/kubepi/internal/service/v1/clusterbinding"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	"github.com/KubeOperator/kubepi/internal/service/v1/rolebinding"
	pkgV1 "github.com/KubeOperator/kubepi/pkg/api/v1"
	"github.com/KubeOperator/kubepi/pkg/collectons"
	"github.com/asdine/storm/v3"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

type Handler struct {
	groupService          group.Service
	roleBindingService    rolebinding.Service
	clusterBindingService clusterbinding.Service
	clusterService        cluster.Service
}

func NewHandler() *Handler {
	return &Handler{
		groupService:          group.NewService(),
		roleBindingService:    rolebinding.NewService(),
		clusterBindingService: clusterbinding.NewService(),
		clusterService:        cluster.NewService(),
	}
}

// Search User
// @Tags users
// @Summary Search users
// @Description Search users by Condition
// @Accept  json
// @Produce  json
// @Success 200 {object} api.Page
// @Security ApiKeyAuth
// @Router /users/search [post]
func (h *Handler) SearchGroups() iris.Handler {
	return func(ctx *context.Context) {
		pageNum, _ := ctx.Values().GetInt(pkgV1.PageNum)
		pageSize, _ := ctx.Values().GetInt(pkgV1.PageSize)

		//pattern := ctx.URLParam("pattern")
		var conditions commons.SearchConditions
		if err := ctx.ReadJSON(&conditions); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		groups, total, err := h.groupService.Search(pageNum, pageSize, conditions.Conditions, common.DBOptions{})
		if err != nil {
			if !errors.Is(err, storm.ErrNotFound) {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err.Error())
				return
			}
		}
		us := make([]Group, 0)
		for i := range groups {
			bindings, err := h.roleBindingService.GetRoleBindingBySubject(v1Role.Subject{Kind: "Group", Name: groups[i].Name}, common.DBOptions{})
			if err != nil && !errors.As(err, &storm.ErrNotFound) {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err.Error())
				return
			}
			roles := collectons.NewStringSet()
			for i := range bindings {
				roles.Add(bindings[i].RoleRef)
			}
			us = append(us, Group{
				Group: groups[i],
				Roles: roles.ToSlice(),
			})
		}
		ctx.Values().Set("data", pkgV1.Page{Items: us, Total: total})
	}
}

// Create User
// @Tags users
// @Summary Create user
// @Description Create user
// @Accept  json
// @Produce  json
// @Param request body docs.UserCreate true "request"
// @Success 200 {object} v1User.User
// @Security ApiKeyAuth
// @Router /users [post]
func (h *Handler) CreateGroup() iris.Handler {
	return func(ctx *context.Context) {
		var req Group
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		u := ctx.Values().Get("profile")
		profile := u.(session.UserProfile)
		//tx
		tx, err := server.DB().Begin(true)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		//创建人
		req.CreatedBy = profile.Name
		//写入数据库
		if err := h.groupService.Create(&req.Group, common.DBOptions{DB: tx}); err != nil {
			_ = tx.Rollback()
			if errors.Is(err, storm.ErrAlreadyExists) {
				ctx.Values().Set("message", "groupname already exists")
				return
			}
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		if len(req.Roles) > 0 {
			for i := range req.Roles {
				roleName := req.Roles[i]
				binding := v1Role.Binding{
					BaseModel: v1.BaseModel{
						Kind:       "RoleBind",
						ApiVersion: "v1",
						CreatedBy:  profile.Name,
					},
					Metadata: v1.Metadata{
						Name: fmt.Sprintf("role-binding-%s-%s", roleName, req.Name),
					},
					Subject: v1Role.Subject{
						Kind: "Group",
						Name: req.Name,
					},
					RoleRef: roleName,
				}
				if err := h.roleBindingService.CreateRoleBinding(&binding, common.DBOptions{DB: tx}); err != nil {
					_ = tx.Rollback()
					ctx.StatusCode(iris.StatusInternalServerError)
					ctx.Values().Set("message", err.Error())
					return
				}
			}
		}
		_ = tx.Commit()
		ctx.Values().Set("data", req)
	}
}

// Delete User
// @Tags users
// @Summary Delete user by name
// @Description Delete user by name
// @Accept  json
// @Produce  json
// @Param name path string true "用户名称"
// @Success 200 {object} v1User.User
// @Security ApiKeyAuth
// @Router /users/{name} [delete]
func (h *Handler) DeleteGroup() iris.Handler {
	return func(ctx *context.Context) {
		groupName := ctx.Params().GetString("name")
		tx, _ := server.DB().Begin(true)
		txOptions := common.DBOptions{DB: tx}

		rbs, err := h.roleBindingService.GetRoleBindingBySubject(v1Role.Subject{
			Kind: "Group",
			Name: groupName,
		}, txOptions)
		if err != nil && !errors.As(err, &storm.ErrNotFound) {
			_ = tx.Rollback()
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		for i := range rbs {
			if err := h.roleBindingService.Delete(rbs[i].Name, txOptions); err != nil {
				_ = tx.Rollback()
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err.Error())
				return
			}
		}
		cbs, err := h.clusterBindingService.GetBindingsByUserName(groupName, txOptions)
		if err != nil && !errors.As(err, &storm.ErrNotFound) {
			_ = tx.Rollback()
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}

		for i := range cbs {
			c, err := h.clusterService.Get(cbs[i].ClusterRef, common.DBOptions{})
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", fmt.Sprintf("get cluster failed: %s", err.Error()))
				return
			}
			k := kubernetes.NewKubernetes(c)
			if err := k.CleanManagedGroupClusterRoleBinding(cbs[i].UserRef); err != nil {
				server.Logger().Errorf("can not delete cluster member %s : %s", cbs[i].UserRef, err)
			}
			if err := k.CleanManagedGroupRoleBinding(cbs[i].UserRef); err != nil {
				server.Logger().Errorf("can not delete cluster member %s : %s", cbs[i].UserRef, err)
			}

			if err := h.clusterBindingService.Delete(cbs[i].Name, txOptions); err != nil {
				_ = tx.Rollback()
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err.Error())
				return
			}
		}
		if err := h.groupService.Delete(groupName, txOptions); err != nil {
			_ = tx.Rollback()
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		_ = tx.Commit()
	}
}

// Get User
// @Tags users
// @Summary Get user by name
// @Description Get user by name
// @Accept  json
// @Produce  json
// @Param name path string true "用户名称"
// @Success 200 {object} v1User.User
// @Security ApiKeyAuth
// @Router /users/{name} [get]
func (h *Handler) GetGroup() iris.Handler {
	return func(ctx *context.Context) {
		groupName := ctx.Params().GetString("name")
		u, err := h.groupService.GetByName(groupName, common.DBOptions{})
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		bindings, err := h.roleBindingService.GetRoleBindingBySubject(v1Role.Subject{Kind: "Group", Name: u.Name}, common.DBOptions{})
		if err != nil && !errors.As(err, &storm.ErrNotFound) {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		roles := collectons.NewStringSet()
		for i := range bindings {
			roles.Add(bindings[i].RoleRef)
		}
		ctx.Values().Set("data", &Group{Group: *u, Roles: roles.ToSlice()})
	}
}

// // List User
// // @Tags users
// // @Summary List all users
// // @Description List all users
// // @Accept  json
// // @Produce  json
// // @Success 200 {object} []v1User.User
// // @Security ApiKeyAuth
// // @Router /users [get]
func (h *Handler) GetGroups() iris.Handler {
	return func(ctx *context.Context) {
		us, err := h.groupService.List(common.DBOptions{})
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		ctx.Values().Set("data", us)
	}
}

// Update User
// @Tags users
// @Summary Update user by name
// @Description Update user by name
// @Accept  json
// @Produce  json
// @Param name path string true "用户名称"
// @Success 200 {object} v1User.User
// @Security ApiKeyAuth
// @Router /users/{name} [put]
func (h *Handler) UpdateGroup() iris.Handler {
	return func(ctx *context.Context) {
		groupName := ctx.Params().GetString("name")
		var req Group
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", err.Error())
			return
		}
		u := ctx.Values().Get("profile")
		profile := u.(session.UserProfile)
		tx, err := server.DB().Begin(true)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		if err := h.groupService.Update(groupName, &req.Group, common.DBOptions{DB: tx}); err != nil {
			_ = tx.Rollback()
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		bindings, err := h.roleBindingService.GetRoleBindingBySubject(v1Role.Subject{Kind: "Group", Name: groupName}, common.DBOptions{})
		if err != nil && !errors.As(err, &storm.ErrNotFound) {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err.Error())
			return
		}
		currentRoles := collectons.NewStringSet()
		for i := range bindings {
			currentRoles.Add(bindings[i].RoleRef)
		}
		for i := range req.Roles {
			r := req.Roles[i]
			if currentRoles.Exists(r) {
				continue
			}
			binding := v1Role.Binding{
				BaseModel: v1.BaseModel{
					Kind:       "RoleBind",
					ApiVersion: "v1",
					CreatedBy:  profile.Name,
				},
				Metadata: v1.Metadata{
					Name: fmt.Sprintf("role-binding-%s-%s", r, req.Name),
				},
				Subject: v1Role.Subject{
					Kind: "Group",
					Name: req.Name,
				},
				RoleRef: r,
			}
			if err := h.roleBindingService.CreateRoleBinding(&binding, common.DBOptions{DB: tx}); err != nil {
				_ = tx.Rollback()
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err.Error())
				return
			}
			currentRoles.Add(binding.RoleRef)
		}
		diffs := currentRoles.Difference(req.Roles)

		for i := range bindings {
			for j := range diffs {
				if bindings[i].RoleRef == diffs[j] {
					if err := h.roleBindingService.Delete(bindings[i].Name, common.DBOptions{DB: tx}); err != nil {
						_ = tx.Rollback()
						ctx.StatusCode(iris.StatusInternalServerError)
						ctx.Values().Set("message", err.Error())
						return
					}
				}
			}
		}

		_ = tx.Commit()
		ctx.Values().Set("data", &req)
	}
}

func Install(parent iris.Party) {
	handler := NewHandler()
	sp := parent.Party("/groups")
	sp.Post("/search", handler.SearchGroups())
	sp.Post("/", handler.CreateGroup())
	sp.Delete("/:name", handler.DeleteGroup())
	sp.Get("/:name", handler.GetGroup())
	sp.Put("/:name", handler.UpdateGroup())
	sp.Get("/", handler.GetGroups())
}
