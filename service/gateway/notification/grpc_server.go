package notification

import (
	"context"
	"fmt"
	"log"
	"strings"

	pb "reservation-sys/service/gateway/api/gen/notification"

	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer 通知服务的 gRPC 实现
type GRPCServer struct {
	pb.UnimplementedNotificationServiceServer
	oa         *officialaccount.OfficialAccount
	templateID string
}

// NewGRPCServer 创建通知 gRPC 服务端
func NewGRPCServer(oa *officialaccount.OfficialAccount, templateID string) *GRPCServer {
	return &GRPCServer{
		oa:         oa,
		templateID: templateID,
	}
}

// SendApprovalNotification 发送审核通过通知（gRPC 方法）。
// 由 Admin 服务调用，向用户推送微信模板消息，通知预约已审核通过。
//
// 参数:
//   - ctx: gRPC 上下文
//   - req: 通知请求（openid + 申请人信息 + 时段列表 + 订单号）
//
// 返回值:
//   - *pb.NotificationResp: 发送结果（success + message）
//   - error: 微信服务号未初始化、模板 ID 未配置、发送失败时返回 gRPC 错误
func (s *GRPCServer) SendApprovalNotification(ctx context.Context, req *pb.ApprovalNotificationReq) (*pb.NotificationResp, error) {
	if s.oa == nil {
		return nil, status.Error(codes.FailedPrecondition, "微信服务号未初始化")
	}
	if s.templateID == "" {
		return nil, status.Error(codes.FailedPrecondition, "微信模板消息ID未配置")
	}

	// 构建时段与密码信息
	slotParts := make([]string, 0, len(req.Slots))
	for _, s := range req.Slots {
		line := fmt.Sprintf("%s~%s", s.StartTime, s.EndTime)
		if s.Password != "" {
			line += fmt.Sprintf(" 密码:%s", s.Password)
		}
		slotParts = append(slotParts, line)
	}
	slotsText := strings.Join(slotParts, "\n")

	tplMsg := &message.TemplateMessage{
		ToUser:     req.Openid,
		TemplateID: s.templateID,
		Data: map[string]*message.TemplateDataItem{
			"first": {
				Value: "您的场地预约已审核通过！\n",
				Color: "#10B981",
			},
			"keyword1": {
				Value: req.ApplicantName,
			},
			"keyword2": {
				Value: slotsText,
			},
			"keyword3": {
				Value: req.AlumniAssociation,
			},
			"remark": {
				Value: fmt.Sprintf("\n订单号: %s\n请凭门锁密码在预约时间段内使用场地。", req.OrderNo),
			},
		},
	}

	msgID, err := s.oa.GetTemplate().Send(tplMsg)
	if err != nil {
		log.Printf("[error][notification/grpc] 发送模板消息失败: order_no=%s openid=%s err=%v", req.OrderNo, req.Openid, err)
		return nil, status.Errorf(codes.Internal, "发送微信通知失败: %v", err)
	}

	log.Printf("[info][notification/grpc] 模板消息发送成功: order_no=%s openid=%s msgid=%d", req.OrderNo, req.Openid, msgID)
	return &pb.NotificationResp{Success: true, Message: fmt.Sprintf("通知已发送(msgid=%d)", msgID)}, nil
}

// SendRejectionNotification 发送审核驳回通知（gRPC 方法）。
// 由 Admin 服务调用，向用户推送微信模板消息，通知预约未通过审核。
//
// 参数:
//   - ctx: gRPC 上下文
//   - req: 驳回通知请求（openid + 申请人信息 + 时段列表 + 驳回原因）
//
// 返回值:
//   - *pb.NotificationResp: 发送结果（success + message）
//   - error: 微信服务号未初始化、模板 ID 未配置、发送失败时返回 gRPC 错误
func (s *GRPCServer) SendRejectionNotification(ctx context.Context, req *pb.RejectionNotificationReq) (*pb.NotificationResp, error) {
	if s.oa == nil {
		return nil, status.Error(codes.FailedPrecondition, "微信服务号未初始化")
	}
	if s.templateID == "" {
		return nil, status.Error(codes.FailedPrecondition, "微信模板消息ID未配置")
	}

	slotParts := make([]string, 0, len(req.Slots))
	for _, s := range req.Slots {
		slotParts = append(slotParts, fmt.Sprintf("%s~%s", s.StartTime, s.EndTime))
	}
	slotsText := strings.Join(slotParts, "\n")

	reason := req.Reason
	if reason == "" {
		reason = "请咨询管理员了解详情"
	}

	tplMsg := &message.TemplateMessage{
		ToUser:     req.Openid,
		TemplateID: s.templateID,
		Data: map[string]*message.TemplateDataItem{
			"first": {
				Value: "您的场地预约未通过审核。\n",
				Color: "#EF4444",
			},
			"keyword1": {
				Value: req.ApplicantName,
			},
			"keyword2": {
				Value: slotsText,
			},
			"keyword3": {
				Value: req.AlumniAssociation,
			},
			"remark": {
				Value: fmt.Sprintf("\n驳回原因: %s\n如有疑问请联系管理员。", reason),
			},
		},
	}

	msgID, err := s.oa.GetTemplate().Send(tplMsg)
	if err != nil {
		log.Printf("[error][notification/grpc] 发送驳回通知失败: order_no=%s openid=%s err=%v", req.OrderNo, req.Openid, err)
		return nil, status.Errorf(codes.Internal, "发送微信通知失败: %v", err)
	}

	log.Printf("[info][notification/grpc] 驳回通知发送成功: order_no=%s openid=%s msgid=%d", req.OrderNo, req.Openid, msgID)
	return &pb.NotificationResp{Success: true, Message: fmt.Sprintf("通知已发送(msgid=%d)", msgID)}, nil
}
