package group

import v1Group "github.com/KubeOperator/kubepi/internal/model/v1/group"

type Group struct {
	v1Group.Group
	Roles       []string `json:"roles"`
	OldPassword string   `json:"oldPassword"`
	Password    string   `json:"password"`
}
