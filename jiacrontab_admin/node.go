package admin

import (
	"jiacrontab/models"
	"jiacrontab/pkg/proto"
	"jiacrontab/pkg/rpc"

	"github.com/kataras/iris"
)

// GetnodeList 获得任务节点列表
// 超级管理员获得全部节点
// 普通用户获得分组列表
func getNodeList(c iris.Context) {
	var (
		ctx      = wrapCtx(c)
		err      error
		nodeList []models.Node
		reqBody  GetNodeListReqParams
		groupID  uint
		count    int
	)
	if groupID, err = ctx.getGroupIDFromToken(); err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	if err = reqBody.verify(ctx); err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	if groupID == 0 {
		err = models.DB().Offset(reqBody.Page - 1).Limit(reqBody.Pagesize).Find(&nodeList).Error
		models.DB().Model(&models.Node{}).Count(&count)
	} else {
		err = models.DB().Where("group_id=?", groupID).Offset(reqBody.Page - 1).Limit(reqBody.Pagesize).Find(&nodeList).Error
		models.DB().Model(&models.Node{}).Where("group_id=?", groupID).Count(&count)
	}

	if err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	ctx.respSucc("", map[string]interface{}{
		"list":     nodeList,
		"total":    count,
		"page":     reqBody.Page,
		"pagesize": reqBody.Pagesize,
	})
}

func deleteNode(c iris.Context) {
	var (
		err     error
		ctx     = wrapCtx(c)
		reqBody DeleteNodeReqParams
		node    models.Node
		cla     CustomerClaims
	)
	if err = reqBody.verify(ctx); err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	if cla, err = ctx.getClaimsFromToken(); err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	// 普通用户不允许修改其他分组节点信息
	if cla.GroupID != 0 && reqBody.GroupID != cla.GroupID {
		ctx.respError(proto.Code_Error, proto.Msg_NotAllowed, nil)
		return
	}
	// 普通用户不允许修改节点
	if !cla.Root {
		ctx.respError(proto.Code_Error, proto.Msg_NotAllowed, nil)
		return
	}

	if err = node.Delete(reqBody.GroupID, reqBody.Addr); err == nil {
		rpc.DelNode(reqBody.Addr)
		ctx.respError(proto.Code_Error, "删除失败", nil)
		return
	}
	ctx.pubEvent(event_DelNodeDesc, reqBody.Addr, "")
	ctx.respSucc("", nil)
}

func updateNode(c iris.Context) {
	var (
		err     error
		ctx     = wrapCtx(c)
		reqBody UpdateNodeReqParams
		node    models.Node
		cla     CustomerClaims
	)

	if err = reqBody.verify(ctx); err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	if cla, err = ctx.getClaimsFromToken(); err != nil {
		ctx.respError(proto.Code_Error, err.Error(), nil)
		return
	}

	if !cla.Root {
		ctx.respError(proto.Code_Error, proto.Msg_NotAllowed, nil)
		return
	}

	// 普通用户不允许修改其他分组节点信息
	if cla.GroupID != 0 && reqBody.GroupID != cla.GroupID {
		ctx.respError(proto.Code_Error, proto.Msg_NotAllowed, nil)
		return
	}

	node.Name = reqBody.Name

	if err = node.Rename(reqBody.GroupID, reqBody.Addr); err == nil {
		ctx.respError(proto.Code_Error, "更新失败", err)
		return
	}

	ctx.pubEvent(event_RenameNode, reqBody.Addr, reqBody)
	ctx.respSucc("", nil)
}
