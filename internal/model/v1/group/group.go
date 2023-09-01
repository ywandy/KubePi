package group

import v1 "github.com/KubeOperator/kubepi/internal/model/v1"

type Group struct {
	v1.BaseModel `storm:"inline"`
	v1.Metadata  `storm:"inline"`
	NickName     string `json:"nickName" storm:"index"`
}