package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/new-api-tools/backend/internal/auth"
	"github.com/new-api-tools/backend/internal/models"
	"github.com/new-api-tools/backend/internal/service"
)

type agentLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type agentSettingsRequest struct {
	AnnouncementHTML         string  `json:"announcement_html"`
	DefaultCommissionRate    float64 `json:"default_commission_rate"`
	SecondCommissionEnabled  bool    `json:"second_commission_enabled"`
	SecondCommissionRate     float64 `json:"second_commission_rate"`
	RedemptionQuotaUnit      float64 `json:"redemption_quota_unit"`
	RedemptionCommissionUnit float64 `json:"redemption_commission_unit"`
}

type agentRateRequest struct {
	UserID                  int64   `json:"user_id" binding:"required"`
	CommissionRate          float64 `json:"commission_rate"`
	SecondCommissionEnabled bool    `json:"second_commission_enabled"`
	SecondCommissionRate    float64 `json:"second_commission_rate"`
}

type agentSettlementRequest struct {
	UserID        int64   `json:"user_id" binding:"required"`
	PeriodType    string  `json:"period_type"`
	PeriodStart   string  `json:"period_start" binding:"required"`
	PeriodEnd     string  `json:"period_end" binding:"required"`
	SettledAmount float64 `json:"settled_amount" binding:"required"`
	Note          string  `json:"note"`
}

type agentSettlementStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type agentPowerRequest struct {
	PowerAgent bool `json:"power_agent"`
}

func RegisterAgentPortalRoutes(r *gin.RouterGroup) {
	g := r.Group("/agent")
	{
		g.POST("/auth/login", AgentLogin)
		g.POST("/auth/logout", AgentLogout)
		g.GET("/me", AgentMe)
		g.GET("/summary", AgentSummary)
		g.GET("/invitees", AgentInvitees)
		g.GET("/invitee-ranking", AgentInviteeRanking)
		g.GET("/invitee-trend", AgentInviteeTrend)
		g.GET("/settlement-stats", AgentSettlementStats)
		g.POST("/commission-redemptions/:id/redeem", AgentRedeemCommissionCode)
		g.GET("/top-ups", AgentTopUps)
	}
}

func RegisterAgentAdminRoutes(r *gin.RouterGroup) {
	g := r.Group("/agent-admin")
	{
		g.GET("/settings", AgentAdminSettings)
		g.GET("/commission-stats", AgentAdminCommissionStats)
		g.GET("/settlements", AgentAdminSettlementRecords)
		g.GET("/commission-redemptions", AgentAdminCommissionRedemptionRecords)
		g.POST("/settlements", AgentCreateCommissionSettlement)
		g.POST("/settlements/:id/status", AgentUpdateSettlementStatus)
		g.DELETE("/settlements/:id", AgentDeleteSettlement)
		g.POST("/settings", AgentSaveAdminSettings)
		g.POST("/rates", AgentSaveCommissionRate)
		g.DELETE("/rates/:user_id", AgentDeleteCommissionRate)
		g.POST("/power-agents/:user_id", AgentSetPowerAgent)
	}
}

func AgentLogin(c *gin.Context) {
	var req agentLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "请求格式错误", err.Error()))
		return
	}

	agent, err := service.NewAgentService().Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "账号或密码错误"})
		return
	}

	token, expiresAt, err := auth.GenerateToken("agent:" + strconv.FormatInt(agent.ID, 10))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("TOKEN_ERROR", "Token 生成失败", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "登录成功",
		"token":      token,
		"expires_at": expiresAt.Format(time.RFC3339),
		"data":       agent,
	})
}

func AgentLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已登出"})
}

func AgentMe(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	agent, err := service.NewAgentService().GetAgent(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResp("UNAUTHORIZED", "代理账号不存在或已删除", ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": agent})
}

func AgentSummary(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	data, err := service.NewAgentService().Summary(userID, c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentInvitees(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	data, err := service.NewAgentService().Invitees(userID, page, pageSize, c.Query("keyword"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentTopUps(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	data, err := service.NewAgentService().TopUps(userID, page, pageSize, c.Query("status"), c.Query("keyword"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentInviteeRanking(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	data, err := service.NewAgentService().InviteeRanking(userID, days, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentInviteeTrend(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	data, err := service.NewAgentService().InviteeTrend(userID, c.DefaultQuery("mode", "day"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentSettlementStats(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	data, err := service.NewAgentService().AgentSettlementStats(userID, c.DefaultQuery("mode", "day"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentRedeemCommissionCode(c *gin.Context) {
	userID, ok := currentAgentID(c)
	if !ok {
		return
	}
	redemptionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid redemption id", ""))
		return
	}
	data, err := service.NewAgentService().RedeemCommissionCode(userID, redemptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("REDEEM_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentAdminSettings(c *gin.Context) {
	data, err := service.NewAgentService().AdminSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentAdminCommissionStats(c *gin.Context) {
	data, err := service.NewAgentService().AdminCommissionStats(c.DefaultQuery("mode", "day"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentCreateCommissionSettlement(c *gin.Context) {
	var req agentSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "请求格式错误", err.Error()))
		return
	}
	err := service.NewAgentService().CreateCommissionSettlement(service.AdminSettlementRequest{
		UserID:        req.UserID,
		PeriodType:    req.PeriodType,
		PeriodStart:   req.PeriodStart,
		PeriodEnd:     req.PeriodEnd,
		SettledAmount: req.SettledAmount,
		Note:          req.Note,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("SAVE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AgentAdminSettlementRecords(c *gin.Context) {
	data, err := service.NewAgentService().AdminSettlementRecords(c.DefaultQuery("mode", "all"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentAdminCommissionRedemptionRecords(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	data, err := service.NewAgentService().AdminCommissionRedemptionRecords(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("QUERY_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func AgentUpdateSettlementStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid settlement id", ""))
		return
	}
	var req agentSettlementStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "请求格式错误", err.Error()))
		return
	}
	if err := service.NewAgentService().UpdateSettlementStatus(id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("SAVE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AgentDeleteSettlement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid settlement id", ""))
		return
	}
	if err := service.NewAgentService().DeleteSettlement(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("DELETE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AgentSaveAdminSettings(c *gin.Context) {
	var req agentSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "请求格式错误", err.Error()))
		return
	}
	if err := service.NewAgentService().SaveAdminSettings(req.AnnouncementHTML, req.DefaultCommissionRate, req.SecondCommissionEnabled, req.SecondCommissionRate, req.RedemptionQuotaUnit, req.RedemptionCommissionUnit); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("SAVE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AgentSaveCommissionRate(c *gin.Context) {
	var req agentRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "请求格式错误", err.Error()))
		return
	}
	if err := service.NewAgentService().SaveCommissionRate(req.UserID, req.CommissionRate, req.SecondCommissionEnabled, req.SecondCommissionRate); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("SAVE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AgentDeleteCommissionRate(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid user id", ""))
		return
	}
	if err := service.NewAgentService().DeleteCommissionRate(userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("DELETE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AgentSetPowerAgent(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid user id", ""))
		return
	}
	var req agentPowerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "请求格式错误", err.Error()))
		return
	}
	if err := service.NewAgentService().SetPowerAgent(userID, req.PowerAgent); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("SAVE_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func currentAgentID(c *gin.Context) (int64, bool) {
	sub, _ := c.Get("user_sub")
	subject, _ := sub.(string)
	userID, ok := service.ParseAgentSubject(subject)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResp("UNAUTHORIZED", "Agent token required", ""))
		return 0, false
	}
	return userID, true
}
