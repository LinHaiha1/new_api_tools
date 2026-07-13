package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/new-api-tools/backend/internal/models"
	"github.com/new-api-tools/backend/internal/service"
)

func RegisterPromptAuditRoutes(r *gin.RouterGroup) {
	g := r.Group("/prompt-audit")
	g.GET("/events", ListPromptAuditEvents)
	g.GET("/config", GetPromptAuditConfig)
	g.PUT("/config", UpdatePromptAuditConfig)
	g.POST("/config", UpdatePromptAuditConfig)
}

func RegisterPromptAuditInternalRoutes(r *gin.Engine) {
	g := r.Group("/api/internal/prompt-audit")
	g.POST("/events", ReceivePromptAuditEvent)
	g.GET("/config", GetPromptAuditInternalConfig)
	g.GET("/config-version", GetPromptAuditInternalConfigVersion)
}

func ListPromptAuditEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	userID, _ := strconv.Atoi(c.DefaultQuery("user_id", "0"))

	data, err := service.ListPromptAuditEvents(limit, offset, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("PROMPT_AUDIT_LIST_FAILED", err.Error(), ""))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func ReceivePromptAuditEvent(c *gin.Context) {
	if !service.PromptAuditSecretConfigured() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResp("PROMPT_AUDIT_NOT_CONFIGURED", "PROMPT_AUDIT_SECRET is not configured", ""))
		return
	}
	if !service.VerifyPromptAuditSecret(c.GetHeader("X-Prompt-Audit-Secret")) {
		c.JSON(http.StatusUnauthorized, models.ErrorResp("UNAUTHORIZED", "Invalid prompt audit secret", ""))
		return
	}

	var event service.PromptAuditEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid request body", err.Error()))
		return
	}

	if err := service.SavePromptAuditEvent(event); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("PROMPT_AUDIT_SAVE_FAILED", err.Error(), ""))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"stored":     true,
			"request_id": event.RequestID,
		},
	})
}

func GetPromptAuditConfig(c *gin.Context) {
	cfg, err := service.GetPromptAuditConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("PROMPT_AUDIT_CONFIG_LOAD_FAILED", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg,
	})
}

func UpdatePromptAuditConfig(c *gin.Context) {
	var input service.PromptAuditConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp("INVALID_PARAMS", "Invalid request body", err.Error()))
		return
	}
	cfg, err := service.SavePromptAuditConfig(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("PROMPT_AUDIT_CONFIG_SAVE_FAILED", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg,
	})
}

func GetPromptAuditInternalConfig(c *gin.Context) {
	if !service.PromptAuditSecretConfigured() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResp("PROMPT_AUDIT_NOT_CONFIGURED", "PROMPT_AUDIT_SECRET is not configured", ""))
		return
	}
	if !service.VerifyPromptAuditSecret(c.GetHeader("X-Prompt-Audit-Secret")) {
		c.JSON(http.StatusUnauthorized, models.ErrorResp("UNAUTHORIZED", "Invalid prompt audit secret", ""))
		return
	}
	cfg, err := service.GetPromptAuditConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("PROMPT_AUDIT_CONFIG_LOAD_FAILED", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg,
	})
}

func GetPromptAuditInternalConfigVersion(c *gin.Context) {
	if !service.PromptAuditSecretConfigured() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResp("PROMPT_AUDIT_NOT_CONFIGURED", "PROMPT_AUDIT_SECRET is not configured", ""))
		return
	}
	if !service.VerifyPromptAuditSecret(c.GetHeader("X-Prompt-Audit-Secret")) {
		c.JSON(http.StatusUnauthorized, models.ErrorResp("UNAUTHORIZED", "Invalid prompt audit secret", ""))
		return
	}
	cfg, err := service.GetPromptAuditConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("PROMPT_AUDIT_CONFIG_LOAD_FAILED", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":    cfg.Enabled,
			"version":    cfg.Version,
			"updated_at": cfg.UpdatedAt,
			"count":      len(cfg.Keywords),
		},
	})
}
