package repositoryservicelogic

import (
	"context"

	"github.com/ChenJustin666/axiops/application/console-rpc/internal/repositorymanager/types"
	"github.com/ChenJustin666/axiops/application/console-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/console-rpc/pb"
	"github.com/ChenJustin666/axiops/common/handler/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateUserLogic) UpdateUser(in *pb.UpdateUserReq) (*pb.UpdateUserResp, error) {
	client, err := l.svcCtx.HarborManager.Get(in.RegistryUuid)
	if err != nil {
		return nil, errorx.Msg("获取仓库客户端失败")
	}

	req := &types.UserUpdateReq{
		Email:    in.Email,
		Realname: in.Realname,
		Comment:  in.Comment,
	}

	err = client.User().Update(in.UserId, req)
	if err != nil {
		return nil, errorx.Msg("更新用户失败: " + err.Error())
	}

	return &pb.UpdateUserResp{Message: "更新成功"}, nil
}
