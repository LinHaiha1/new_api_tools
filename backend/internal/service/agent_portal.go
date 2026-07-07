package service

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/new-api-tools/backend/internal/config"
	"github.com/new-api-tools/backend/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type AgentService struct {
	db *database.Manager
}

type AgentUser struct {
	ID          int64       `json:"id" db:"id"`
	Username    string      `json:"username" db:"username"`
	DisplayName interface{} `json:"display_name" db:"display_name"`
	AffCode     interface{} `json:"aff_code" db:"aff_code"`
	InviteURL   string      `json:"invite_url" db:"-"`
	AffCount    int64       `json:"aff_count" db:"aff_count"`
	AffQuota    int64       `json:"aff_quota" db:"aff_quota"`
	AffHistory  int64       `json:"aff_history" db:"aff_history"`
}

type AgentInvitee struct {
	ID           int64       `json:"id" db:"id"`
	Username     interface{} `json:"username" db:"username"`
	DisplayName  interface{} `json:"display_name" db:"display_name"`
	Email        interface{} `json:"email" db:"email"`
	Status       int64       `json:"status" db:"status"`
	Quota        int64       `json:"quota" db:"quota"`
	UsedQuota    int64       `json:"used_quota" db:"used_quota"`
	RequestCount int64       `json:"request_count" db:"request_count"`
	CreatedAt    interface{} `json:"created_at" db:"created_at"`
	LastLoginAt  int64       `json:"last_login_at" db:"last_login_at"`
	TopUpCount   int64       `json:"topup_count" db:"topup_count"`
	SuccessMoney float64     `json:"success_money" db:"success_money"`
	LastTopUpAt  interface{} `json:"last_topup_at" db:"last_topup_at"`
	SubInvitees  int64       `json:"sub_invitees" db:"sub_invitees"`
	SubQualified int64       `json:"sub_qualified_invitees" db:"sub_qualified_invitees"`
	SubMoney     float64     `json:"sub_success_money" db:"sub_success_money"`
	Commission   float64     `json:"commission_rate" db:"commission_rate"`
	SecondRate   float64     `json:"second_commission_rate" db:"second_commission_rate"`
	SecondMoney  float64     `json:"second_commission_estimate" db:"second_commission_estimate"`
}

type AgentRankUser struct {
	UserID          int64       `json:"user_id" db:"user_id"`
	Username        interface{} `json:"username" db:"username"`
	DisplayName     interface{} `json:"display_name" db:"display_name"`
	SuccessCount    int64       `json:"success_count" db:"success_count"`
	SuccessMoney    float64     `json:"success_money" db:"success_money"`
	SubInvitees     int64       `json:"sub_invitees" db:"sub_invitees"`
	SubSuccessMoney float64     `json:"sub_success_money" db:"sub_success_money"`
}

type AgentTopUp struct {
	ID              int64       `json:"id" db:"id"`
	UserID          int64       `json:"user_id" db:"user_id"`
	Username        interface{} `json:"username" db:"username"`
	DisplayName     interface{} `json:"display_name" db:"display_name"`
	Money           float64     `json:"money" db:"money"`
	Amount          int64       `json:"amount" db:"amount"`
	TradeNo         interface{} `json:"trade_no" db:"trade_no"`
	PaymentMethod   interface{} `json:"payment_method" db:"payment_method"`
	PaymentProvider interface{} `json:"payment_provider" db:"payment_provider"`
	CreateTime      interface{} `json:"create_time" db:"create_time"`
	CompleteTime    interface{} `json:"complete_time" db:"complete_time"`
	RawStatus       string      `json:"raw_status" db:"raw_status"`
	StatusBucket    string      `json:"status_bucket" db:"status_bucket"`
}

type AdminSettlementRequest struct {
	UserID        int64   `json:"user_id"`
	PeriodType    string  `json:"period_type"`
	PeriodStart   string  `json:"period_start"`
	PeriodEnd     string  `json:"period_end"`
	SettledAmount float64 `json:"settled_amount"`
	Note          string  `json:"note"`
}

type AdminSettlementStatusRequest struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func NewAgentService() *AgentService {
	return &AgentService{db: database.Get()}
}

func topUpBucketSQL(column string) string {
	trimmed := fmt.Sprintf("TRIM(COALESCE(%s, ''))", column)
	lower := fmt.Sprintf("LOWER(%s)", trimmed)
	return fmt.Sprintf(`CASE
		WHEN %s = '' THEN 'pending'
		WHEN %s IN ('success', 'completed') OR %s = '1' THEN 'success'
		WHEN %s IN ('failed', 'error') OR %s = '-1' THEN 'failed'
		WHEN %s = 'expired' THEN 'expired'
		WHEN %s IN ('pending', 'processing', 'created', 'waiting', 'unpaid') OR %s = '0' THEN 'pending'
		ELSE 'unknown'
	END`, trimmed, lower, trimmed, lower, trimmed, lower, lower, trimmed)
}

func (s *AgentService) Authenticate(username, password string) (*AgentUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, fmt.Errorf("账号或密码错误")
	}

	var row struct {
		AgentUser
		Password string `db:"password"`
	}
	query := s.db.RebindQuery(`
		SELECT id, username, display_name, aff_code, aff_count, aff_quota, aff_history, password
		FROM users
		WHERE username = ? AND deleted_at IS NULL
		LIMIT 1`)
	if err := s.db.DB.Get(&row, query, username); err != nil {
		return nil, fmt.Errorf("账号或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("账号或密码错误")
	}
	row.AgentUser.InviteURL = buildAgentInviteURL(row.AgentUser.AffCode)
	return &row.AgentUser, nil
}

func (s *AgentService) GetAgent(userID int64) (*AgentUser, error) {
	var user AgentUser
	query := s.db.RebindQuery(`
		SELECT id, username, display_name, aff_code, aff_count, aff_quota, aff_history
		FROM users
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1`)
	if err := s.db.DB.Get(&user, query, userID); err != nil {
		return nil, err
	}
	user.AffCode = decodeAgentAffCode(user.AffCode)
	user.InviteURL = buildAgentInviteURL(user.AffCode)
	return &user, nil
}

func (s *AgentService) Summary(userID int64, startDate, endDate string) (map[string]interface{}, error) {
	bucket := topUpBucketSQL("t.status")
	startTs, endTs, hasRange := parseDateRange(startDate, endDate)
	commissionRate := s.AgentCommissionRate(userID)

	totalRows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT COUNT(*) AS invitee_count,
			SUM(CASE WHEN u.status = 1 THEN 1 ELSE 0 END) AS active_count,
			SUM(CASE WHEN u.status = 2 THEN 1 ELSE 0 END) AS banned_count,
			SUM(CASE WHEN COALESCE(p.success_money, 0) > 10 THEN 1 ELSE 0 END) AS qualified_invitee_count,
			SUM(CASE WHEN COALESCE(p.success_money, 0) <= 10 THEN 1 ELSE 0 END) AS unqualified_invitee_count,
			SUM(CASE WHEN COALESCE(p.success_count, 0) > 0 THEN 1 ELSE 0 END) AS paying_user_count,
			SUM(CASE WHEN COALESCE(p.success_count, 0) >= 2 THEN 1 ELSE 0 END) AS repeat_user_count,
			COALESCE(SUM(p.success_count), 0) AS success_count,
			COALESCE(SUM(p.topup_count), 0) AS topup_count,
			COALESCE(SUM(p.success_money), 0) AS success_money
		FROM users u
		LEFT JOIN (
			SELECT user_id,
				COUNT(*) AS topup_count,
				SUM(CASE WHEN (%s) = 'success' THEN 1 ELSE 0 END) AS success_count,
				SUM(CASE WHEN (%s) = 'success' THEN money ELSE 0 END) AS success_money
			FROM top_ups t
			GROUP BY user_id
		) p ON p.user_id = u.id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL`, bucket, bucket)), userID)
	if err != nil {
		return nil, err
	}
	total := firstRow(totalRows)
	total["commission_rate"] = commissionRate
	total["commission_estimate"] = agentFloat64(total["success_money"]) * commissionRate
	total["repeat_rate"] = agentRatio(agentFloat64(total["repeat_user_count"]), agentFloat64(total["paying_user_count"]))
	total["conversion_rate"] = agentRatio(agentFloat64(total["paying_user_count"]), agentFloat64(total["invitee_count"]))

	periodWhere := "u.inviter_id = ? AND u.deleted_at IS NULL"
	periodArgs := []interface{}{userID}
	if hasRange {
		periodWhere += " AND t.create_time >= ? AND t.create_time <= ?"
		periodArgs = append(periodArgs, startTs, endTs)
	}
	periodRows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT COUNT(*) AS topup_count,
			SUM(CASE WHEN (%s) = 'success' THEN 1 ELSE 0 END) AS success_count,
			COALESCE(SUM(CASE WHEN (%s) = 'success' THEN t.money ELSE 0 END), 0) AS success_money,
			COUNT(DISTINCT CASE WHEN (%s) = 'success' THEN t.user_id END) AS paying_user_count,
			COUNT(DISTINCT CASE WHEN p.period_success_count >= 2 THEN p.user_id END) AS repeat_user_count
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		LEFT JOIN (
			SELECT t2.user_id, COUNT(*) AS period_success_count
			FROM top_ups t2
			JOIN users u2 ON u2.id = t2.user_id
			WHERE u2.inviter_id = ? AND u2.deleted_at IS NULL AND (%s) = 'success'%s
			GROUP BY t2.user_id
		) p ON p.user_id = t.user_id
		WHERE %s`, bucket, bucket, bucket, topUpBucketSQL("t2.status"), dateSQL("t2.create_time", hasRange), periodWhere)), append(periodArgs, periodArgs...)...)
	if err != nil {
		return nil, err
	}
	period := firstRow(periodRows)
	period["commission_rate"] = commissionRate
	period["commission_estimate"] = agentFloat64(period["success_money"]) * commissionRate
	period["repeat_rate"] = agentRatio(agentFloat64(period["repeat_user_count"]), agentFloat64(period["paying_user_count"]))
	dailyCompare, err := s.dailyCompare(userID, commissionRate, bucket)
	if err != nil {
		return nil, err
	}

	recentStart := time.Now().AddDate(0, 0, -9)
	recentStart = time.Date(recentStart.Year(), recentStart.Month(), recentStart.Day(), 0, 0, 0, 0, time.Local)
	recentEnd := time.Now()
	recentEnd = time.Date(recentEnd.Year(), recentEnd.Month(), recentEnd.Day(), 23, 59, 59, 0, time.Local)
	dailyRawRows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT t.create_time, t.money
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL AND (%s) = 'success' AND t.create_time >= ? AND t.create_time <= ?`, bucket)), userID, recentStart.Unix(), recentEnd.Unix())
	if err != nil {
		return nil, err
	}
	dailyRows := aggregateDailyMoneyRows(dailyRawRows, recentStart, 10)

	topUserRows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT u.id AS user_id, u.username, u.display_name,
			COUNT(*) AS success_count,
			COALESCE(SUM(t.money), 0) AS success_money
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL AND (%s) = 'success'%s
		GROUP BY u.id, u.username, u.display_name
		ORDER BY success_money DESC
		LIMIT 10`, bucket, dateSQL("t.create_time", hasRange))), dateArgs(userID, startTs, endTs, hasRange)...)
	if err != nil {
		return nil, err
	}

	funnel := map[string]interface{}{
		"invitee_count":           total["invitee_count"],
		"qualified_invitee_count": total["qualified_invitee_count"],
		"paying_user_count":       total["paying_user_count"],
		"repeat_user_count":       total["repeat_user_count"],
	}

	return map[string]interface{}{
		"total":         total,
		"period":        period,
		"daily_compare": dailyCompare,
		"daily_success": dailyRows,
		"top_users":     topUserRows,
		"funnel":        funnel,
		"settings":      s.PublicSettings(userID),
	}, nil
}

func (s *AgentService) secondCommissionSettings(userID int64) (bool, float64) {
	_ = s.EnsureAgentSettingsTables()
	row, err := s.db.QueryOne(s.db.RebindQuery(`SELECT second_commission_enabled, second_commission_rate FROM agent_commission_rates WHERE user_id = ?`), userID)
	if err != nil || row == nil {
		return false, 0.03
	}
	return agentFloat64(row["second_commission_enabled"]) > 0, fallbackRate(agentFloat64(row["second_commission_rate"]), 0.03)
}

func (s *AgentService) dailyCompare(userID int64, commissionRate float64, bucket string) (map[string]interface{}, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	todayEnd := todayStart + 86400 - 1
	yesterdayStart := todayStart - 86400
	yesterdayEnd := todayStart - 1
	rows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN t.create_time >= ? AND t.create_time <= ? AND (%s) = 'success' THEN t.money ELSE 0 END), 0) AS today_success_money,
			COALESCE(SUM(CASE WHEN t.create_time >= ? AND t.create_time <= ? AND (%s) = 'success' THEN t.money ELSE 0 END), 0) AS yesterday_success_money
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL`, bucket, bucket)), todayStart, todayEnd, yesterdayStart, yesterdayEnd, userID)
	if err != nil {
		return nil, err
	}
	row := firstRow(rows)
	row["today_commission_estimate"] = agentFloat64(row["today_success_money"]) * commissionRate
	row["yesterday_commission_estimate"] = agentFloat64(row["yesterday_success_money"]) * commissionRate
	return row, nil
}

func (s *AgentService) AgentSettlementStats(userID int64, mode string, recordLimit int) (map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	commissionRate := s.AgentCommissionRate(userID)
	bucket := topUpBucketSQL("t.status")
	totalRows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT COALESCE(SUM(CASE WHEN (%s) = 'success' THEN t.money ELSE 0 END), 0) AS total_success_money
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL`, bucket)), userID)
	if err != nil {
		return nil, err
	}
	settledRows, err := s.db.Query(s.db.RebindQuery(`
		SELECT COALESCE(SUM(settled_amount), 0) AS settled_amount, MAX(settled_at) AS last_settled_at
		FROM agent_commission_settlements
		WHERE user_id = ? AND status = 'settled'`), userID)
	if err != nil {
		return nil, err
	}
	redeemedRows, err := s.db.Query(s.db.RebindQuery(`
		SELECT COALESCE(SUM(commission_amount), 0) AS redeemed_amount
		FROM agent_commission_redemptions
		WHERE user_id = ? AND status = 'success'`), userID)
	if err != nil {
		return nil, err
	}
	availableRows, err := s.AvailableCommissionRedemptions(userID)
	if err != nil {
		return nil, err
	}
	redeemRecords, err := s.AgentCommissionRedemptionRecords(userID, 10)
	if err != nil {
		return nil, err
	}
	if recordLimit != 50 && recordLimit != 100 {
		recordLimit = 10
	}
	records, err := s.db.Query(s.db.RebindQuery(`
		SELECT id, period_type, period_start, period_end, settled_amount, settled_at, status, note
		FROM agent_commission_settlements
		WHERE user_id = ?
		ORDER BY settled_at DESC, id DESC
		LIMIT ?`), userID, recordLimit)
	if err != nil {
		return nil, err
	}
	trendRows, err := s.agentCommissionTrend(userID, mode, commissionRate)
	if err != nil {
		return nil, err
	}
	total := firstRow(totalRows)
	settled := firstRow(settledRows)
	redeemed := firstRow(redeemedRows)
	periodCommission := 0.0
	periodSettled := 0.0
	periodPending := 0.0
	for _, row := range trendRows {
		periodCommission += agentFloat64(row["commission_amount"])
		periodSettled += agentFloat64(row["settled_amount"])
		periodPending += agentFloat64(row["pending_amount"])
	}
	totalCommission := agentFloat64(total["total_success_money"]) * commissionRate
	settledAmount := agentFloat64(settled["settled_amount"])
	redeemedAmount := agentFloat64(redeemed["redeemed_amount"])
	pending := totalCommission - settledAmount - redeemedAmount
	if pending < 0 {
		pending = 0
	}
	return map[string]interface{}{
		"summary": map[string]interface{}{
			"period_mode":                normalizedSettlementMode(mode),
			"period_label":               agentSettlementLabel(mode),
			"period_commission_estimate": periodCommission,
			"period_settled_amount":      periodSettled,
			"period_pending_amount":      periodPending,
			"total_success_money":        total["total_success_money"],
			"commission_rate":            commissionRate,
			"total_commission_estimate":  totalCommission,
			"settled_amount":             settledAmount,
			"redeemed_amount":            redeemedAmount,
			"pending_amount":             pending,
			"last_settled_at":            settled["last_settled_at"],
		},
		"trend":                 trendRows,
		"records":               records,
		"available_redemptions": availableRows,
		"redemption_records":    redeemRecords,
	}, nil
}

func (s *AgentService) AvailableCommissionRedemptions(userID int64) ([]map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	quotaUnit, commissionUnit := s.redemptionRatio()
	rows, err := s.db.Query(s.db.RebindQuery(`
		SELECT MIN(id) AS id, name, quota, (? * quota / ?) AS commission_amount, COUNT(*) AS stock, MIN(expired_time) AS expired_time
		FROM redemptions
		WHERE deleted_at IS NULL
			AND LOWER(name) LIKE 'yj%%'
			AND (used_user_id IS NULL OR used_user_id = 0)
			AND (redeemed_time IS NULL OR redeemed_time = 0)
			AND (expired_time IS NULL OR expired_time = 0 OR expired_time >= ?)
		GROUP BY name, quota
		ORDER BY quota ASC, name ASC
		LIMIT 50`), commissionUnit, quotaUnit, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return rows, nil
}

func (s *AgentService) AgentCommissionRedemptionRecords(userID int64, limit int) ([]map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	rows, err := s.db.Query(s.db.RebindQuery(`
		SELECT id, redemption_id, redemption_key, redemption_name, quota, commission_amount, status, created_at
		FROM agent_commission_redemptions
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`), userID, limit)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return rows, nil
}

func (s *AgentService) redemptionRatio() (float64, float64) {
	row, err := s.db.QueryOne(`SELECT redemption_quota_unit, redemption_commission_unit FROM agent_settings WHERE id = 1`)
	if err != nil || row == nil {
		return 5000000, 3
	}
	quotaUnit := agentFloat64(row["redemption_quota_unit"])
	commissionUnit := agentFloat64(row["redemption_commission_unit"])
	if quotaUnit <= 0 {
		quotaUnit = 5000000
	}
	if commissionUnit <= 0 {
		commissionUnit = 3
	}
	return quotaUnit, commissionUnit
}

func (s *AgentService) redemptionCommissionAmount(quota, quotaUnit, commissionUnit float64) float64 {
	if quota <= 0 || quotaUnit <= 0 || commissionUnit <= 0 {
		return 0
	}
	return quota * commissionUnit / quotaUnit
}

func (s *AgentService) RedeemCommissionCode(userID, redemptionID int64) (map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	stats, err := s.AgentSettlementStats(userID, "day", 10)
	if err != nil {
		return nil, err
	}
	available := agentFloat64(firstRow([]map[string]interface{}{stats["summary"].(map[string]interface{})})["pending_amount"])
	kc := "`key`"
	code, err := s.db.QueryOne(s.db.RebindQuery(`
		SELECT name, quota
		FROM redemptions
		WHERE id = ? AND deleted_at IS NULL
			AND LOWER(name) LIKE 'yj%%'
			AND (used_user_id IS NULL OR used_user_id = 0)
			AND (redeemed_time IS NULL OR redeemed_time = 0)
			AND (expired_time IS NULL OR expired_time = 0 OR expired_time >= ?)
		LIMIT 1`), redemptionID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	if code == nil {
		return nil, fmt.Errorf("兑换码不可兑换，仅允许兑换名称以 yj 开头且未使用的兑换码")
	}
	quotaUnit, commissionUnit := s.redemptionRatio()
	cost := s.redemptionCommissionAmount(agentFloat64(code["quota"]), quotaUnit, commissionUnit)
	if cost <= 0 {
		return nil, fmt.Errorf("兑换码额度异常")
	}
	if available < cost {
		return nil, fmt.Errorf("可用佣金不足")
	}
	code, err = s.db.QueryOne(s.db.RebindQuery(fmt.Sprintf(`
		SELECT id, name, %s AS redemption_key, quota, (? * quota / ?) AS commission_amount, expired_time
		FROM redemptions
		WHERE deleted_at IS NULL
			AND LOWER(name) LIKE 'yj%%'
			AND name = ?
			AND quota = ?
			AND (used_user_id IS NULL OR used_user_id = 0)
			AND (redeemed_time IS NULL OR redeemed_time = 0)
			AND (expired_time IS NULL OR expired_time = 0 OR expired_time >= ?)
		ORDER BY id ASC
		LIMIT 1`, kc)), commissionUnit, quotaUnit, fmt.Sprint(code["name"]), int64(agentFloat64(code["quota"])), time.Now().Unix())
	if err != nil {
		return nil, err
	}
	if code == nil {
		return nil, fmt.Errorf("该兑换码库存不足")
	}
	now := time.Now().Unix()
	affected, err := s.db.Execute(s.db.RebindQuery(`
		UPDATE redemptions
		SET used_user_id = ?, redeemed_time = ?
		WHERE id = ? AND deleted_at IS NULL AND LOWER(name) LIKE 'yj%' AND (used_user_id IS NULL OR used_user_id = 0) AND (redeemed_time IS NULL OR redeemed_time = 0)`), userID, now, redemptionID)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("兑换码已被兑换")
	}
	_, err = s.db.Execute(s.db.RebindQuery(`
		INSERT INTO agent_commission_redemptions (user_id, redemption_id, redemption_key, redemption_name, quota, commission_amount, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'success', ?)`), userID, redemptionID, fmt.Sprint(code["redemption_key"]), fmt.Sprint(code["name"]), int64(agentFloat64(code["quota"])), cost, now)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"redemption_key": code["redemption_key"], "name": code["name"], "commission_amount": cost}, nil
}

func agentSettlementLabel(mode string) string {
	switch normalizedSettlementMode(mode) {
	case "week":
		return "周"
	case "month":
		return "月"
	default:
		return "日"
	}
}

func (s *AgentService) agentCommissionTrend(userID int64, mode string, commissionRate float64) ([]map[string]interface{}, error) {
	bucket := topUpBucketSQL("t.status")
	now := time.Now()
	start := now.AddDate(0, 0, -9)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	count := 10
	fillMode := "day"
	if mode == "week" {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		count = 5
		fillMode = "week"
	} else if mode == "month" {
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)
		count = 12
		fillMode = "month"
	}
	commissionRawRows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT t.create_time, t.money
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL AND (%s) = 'success' AND t.create_time >= ?`, bucket)), userID, start.Unix())
	if err != nil {
		return nil, err
	}
	settledRanges, err := s.db.Query(s.db.RebindQuery(`
		SELECT period_start, period_end
		FROM agent_commission_settlements
		WHERE user_id = ? AND status = 'settled'`), userID)
	if err != nil {
		return nil, err
	}
	commissionRows := aggregateCommissionMoneyRows(commissionRawRows, start, count, fillMode, commissionRate)
	rows := fillCommissionTrendRows(mergeCommissionTrendRows(commissionRows, nil), start, count, fillMode)
	applySettledCoverage(rows, settledRanges, fillMode)
	return rows, nil
}

func (s *AgentService) EnsureAgentSettingsTables() error {
	if _, err := s.db.Execute(`CREATE TABLE IF NOT EXISTS agent_settings (
		id BIGINT PRIMARY KEY DEFAULT 1,
		announcement_html TEXT NULL,
		default_commission_rate DECIMAL(10,4) NOT NULL DEFAULT 0.2000,
		second_commission_enabled TINYINT NOT NULL DEFAULT 0,
		second_commission_rate DECIMAL(10,4) NOT NULL DEFAULT 0.0300,
		redemption_quota_unit DECIMAL(18,4) NOT NULL DEFAULT 5000000,
		redemption_commission_unit DECIMAL(18,4) NOT NULL DEFAULT 3,
		updated_at BIGINT NOT NULL DEFAULT 0
	)`); err != nil {
		return err
	}
	_ = s.ensureColumn("agent_settings", "second_commission_enabled", "TINYINT NOT NULL DEFAULT 0")
	_ = s.ensureColumn("agent_settings", "second_commission_rate", "DECIMAL(10,4) NOT NULL DEFAULT 0.0300")
	_ = s.ensureColumn("agent_settings", "redemption_quota_unit", "DECIMAL(18,4) NOT NULL DEFAULT 5000000")
	_ = s.ensureColumn("agent_settings", "redemption_commission_unit", "DECIMAL(18,4) NOT NULL DEFAULT 3")
	if _, err := s.db.Execute(`CREATE TABLE IF NOT EXISTS agent_commission_rates (
		user_id BIGINT PRIMARY KEY,
		commission_rate DECIMAL(10,4) NOT NULL DEFAULT 0.2000,
		second_commission_enabled TINYINT NOT NULL DEFAULT 0,
		second_commission_rate DECIMAL(10,4) NOT NULL DEFAULT 0.0300,
		updated_at BIGINT NOT NULL DEFAULT 0
	)`); err != nil {
		return err
	}
	_ = s.ensureColumn("agent_commission_rates", "second_commission_enabled", "TINYINT NOT NULL DEFAULT 0")
	_ = s.ensureColumn("agent_commission_rates", "second_commission_rate", "DECIMAL(10,4) NOT NULL DEFAULT 0.0300")
	if _, err := s.db.Execute(`CREATE TABLE IF NOT EXISTS agent_commission_redemptions (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		user_id BIGINT NOT NULL,
		redemption_id BIGINT NOT NULL,
		redemption_key VARCHAR(128) NOT NULL,
		redemption_name VARCHAR(255) NOT NULL,
		quota BIGINT NOT NULL DEFAULT 0,
		commission_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
		status VARCHAR(20) NOT NULL DEFAULT 'success',
		created_at BIGINT NOT NULL DEFAULT 0,
		UNIQUE KEY uniq_agent_commission_redemption_id (redemption_id)
	)`); err != nil {
		return err
	}
	if _, err := s.db.Execute(`CREATE TABLE IF NOT EXISTS agent_power_agents (
		user_id BIGINT PRIMARY KEY,
		created_at BIGINT NOT NULL DEFAULT 0
	)`); err != nil {
		return err
	}
	if _, err := s.db.Execute(`CREATE TABLE IF NOT EXISTS agent_commission_settlements (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		user_id BIGINT NOT NULL,
		period_type VARCHAR(20) NOT NULL DEFAULT 'custom',
		period_start VARCHAR(10) NOT NULL,
		period_end VARCHAR(10) NOT NULL,
		settled_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
		note TEXT NULL,
		settled_at BIGINT NOT NULL DEFAULT 0,
		status VARCHAR(20) NOT NULL DEFAULT 'settled',
		created_at BIGINT NOT NULL DEFAULT 0
	)`); err != nil {
		return err
	}
	_ = s.ensureColumn("agent_commission_settlements", "status", "VARCHAR(20) NOT NULL DEFAULT 'settled'")
	_, err := s.db.Execute(`INSERT IGNORE INTO agent_settings (id, announcement_html, default_commission_rate, second_commission_enabled, second_commission_rate, redemption_quota_unit, redemption_commission_unit, updated_at) VALUES (1, '', 0.2000, 0, 0.0300, 5000000, 3, 0)`)
	if err == nil {
		_, _ = s.db.Execute(`UPDATE agent_settings SET default_commission_rate = 0.2000 WHERE id = 1 AND default_commission_rate = 0.1000`)
	}
	return err
}

func (s *AgentService) ensureColumn(table, column, definition string) error {
	var count int
	query := s.db.RebindQuery(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`)
	if err := s.db.DB.Get(&count, query, table, column); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := s.db.Execute(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func (s *AgentService) PublicSettings(userID int64) map[string]interface{} {
	_ = s.EnsureAgentSettingsTables()
	row, err := s.db.QueryOne(s.db.RebindQuery(`
		SELECT s.announcement_html, s.default_commission_rate, s.second_commission_enabled, s.second_commission_rate,
			COALESCE(r.commission_rate, s.default_commission_rate) AS commission_rate,
			COALESCE(r.second_commission_enabled, 0) AS user_second_commission_enabled,
			COALESCE(r.second_commission_rate, s.second_commission_rate) AS user_second_commission_rate
		FROM agent_settings s
		LEFT JOIN agent_commission_rates r ON r.user_id = ?
		WHERE s.id = 1`), userID)
	if err != nil || row == nil {
		return map[string]interface{}{"announcement_html": "", "commission_rate": 0.2, "default_commission_rate": 0.2, "second_commission_enabled": 0, "second_commission_rate": 0.03, "user_second_commission_enabled": 0, "user_second_commission_rate": 0.03}
	}
	return row
}

func (s *AgentService) AgentCommissionRate(userID int64) float64 {
	settings := s.PublicSettings(userID)
	rate := agentFloat64(settings["commission_rate"])
	if rate <= 0 {
		return 0.2
	}
	return rate
}

func (s *AgentService) AdminSettings() (map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	settings, err := s.db.QueryOne(`SELECT announcement_html, default_commission_rate, second_commission_enabled, second_commission_rate, redemption_quota_unit, redemption_commission_unit, updated_at FROM agent_settings WHERE id = 1`)
	if err != nil {
		return nil, err
	}
	rates, err := s.db.Query(`
		SELECT r.user_id, u.username, u.display_name, r.commission_rate, r.second_commission_enabled, r.second_commission_rate, r.updated_at
		FROM agent_commission_rates r
		LEFT JOIN users u ON u.id = r.user_id
		ORDER BY r.updated_at DESC, r.user_id DESC`)
	if err != nil {
		return nil, err
	}
	commissionStats, err := s.AdminCommissionStats("day", "", "")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"settings": firstRow([]map[string]interface{}{settings}), "rates": rates, "commission_stats": commissionStats}, nil
}

func (s *AgentService) AdminCommissionStats(mode, startDate, endDate string) (map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	bucket := topUpBucketSQL("t.status")
	start, end, periodLabel, periodMode := commissionPeriod(mode, startDate, endDate)
	settledRows, err := s.db.Query(s.db.RebindQuery(`
		SELECT user_id, COALESCE(SUM(settled_amount), 0) AS settled_amount, MAX(settled_at) AS last_settled_at
		FROM agent_commission_settlements
		WHERE status = 'settled'
		GROUP BY user_id`))
	if err != nil {
		return nil, err
	}
	settledByUser := map[int64]map[string]interface{}{}
	for _, row := range settledRows {
		settledByUser[int64(agentFloat64(row["user_id"]))] = row
	}
	periodSettledRows, err := s.db.Query(s.db.RebindQuery(`
		SELECT user_id, COALESCE(SUM(settled_amount), 0) AS period_settled_amount
		FROM agent_commission_settlements
		WHERE status = 'settled' AND period_start <= ? AND period_end >= ?
		GROUP BY user_id`), end.Format("2006-01-02"), start.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	periodSettledByUser := map[int64]float64{}
	for _, row := range periodSettledRows {
		periodSettledByUser[int64(agentFloat64(row["user_id"]))] = agentFloat64(row["period_settled_amount"])
	}
	query := s.db.RebindQuery(fmt.Sprintf(`
		SELECT agent.id AS user_id, agent.username, agent.display_name,
			CASE WHEN power.user_id IS NULL THEN 0 ELSE 1 END AS power_agent,
			COALESCE(r.commission_rate, settings.default_commission_rate) AS commission_rate,
			COALESCE(r.second_commission_enabled, 0) AS second_commission_enabled,
			COALESCE(r.second_commission_rate, settings.second_commission_rate) AS second_commission_rate,
			COALESCE(d.invitee_count, 0) AS invitee_count,
			COALESCE(d.period_success_money, 0) AS period_success_money,
			COALESCE(d.period_success_count, 0) AS period_success_count,
			COALESCE(d.total_success_money, 0) AS total_success_money,
			COALESCE(d.total_success_count, 0) AS total_success_count,
			COALESCE(sd.period_second_success_money, 0) AS period_second_success_money,
			COALESCE(sd.total_second_success_money, 0) AS total_second_success_money
		FROM (
			SELECT DISTINCT inviter_id AS user_id FROM users WHERE inviter_id IS NOT NULL AND inviter_id > 0 AND deleted_at IS NULL
			UNION
			SELECT user_id FROM agent_commission_rates
		) candidates
		JOIN users agent ON agent.id = candidates.user_id AND agent.deleted_at IS NULL
		JOIN agent_settings settings ON settings.id = 1
		LEFT JOIN agent_commission_rates r ON r.user_id = agent.id
		LEFT JOIN agent_power_agents power ON power.user_id = agent.id
		LEFT JOIN (
			SELECT u.inviter_id AS agent_id,
				COUNT(DISTINCT u.id) AS invitee_count,
				SUM(CASE WHEN (%s) = 'success' AND t.create_time >= ? AND t.create_time <= ? THEN t.money ELSE 0 END) AS period_success_money,
				SUM(CASE WHEN (%s) = 'success' AND t.create_time >= ? AND t.create_time <= ? THEN 1 ELSE 0 END) AS period_success_count,
				SUM(CASE WHEN (%s) = 'success' THEN t.money ELSE 0 END) AS total_success_money,
				SUM(CASE WHEN (%s) = 'success' THEN 1 ELSE 0 END) AS total_success_count
			FROM users u
			LEFT JOIN top_ups t ON t.user_id = u.id
			WHERE u.inviter_id IS NOT NULL AND u.inviter_id > 0 AND u.deleted_at IS NULL
			GROUP BY u.inviter_id
		) d ON d.agent_id = agent.id
		LEFT JOIN (
			SELECT direct.inviter_id AS agent_id,
				SUM(CASE WHEN (%s) = 'success' AND t.create_time >= ? AND t.create_time <= ? THEN t.money ELSE 0 END) AS period_second_success_money,
				SUM(CASE WHEN (%s) = 'success' THEN t.money ELSE 0 END) AS total_second_success_money
			FROM users direct
			JOIN users child ON child.inviter_id = direct.id AND child.deleted_at IS NULL
			LEFT JOIN top_ups t ON t.user_id = child.id
			WHERE direct.inviter_id IS NOT NULL AND direct.inviter_id > 0 AND direct.deleted_at IS NULL
			GROUP BY direct.inviter_id
		) sd ON sd.agent_id = agent.id
		ORDER BY period_success_money DESC, total_success_money DESC, agent.id DESC`, bucket, bucket, bucket, bucket, bucket, bucket))
	rows, err := s.db.Query(query, start.Unix(), end.Unix(), start.Unix(), end.Unix(), start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	summary := map[string]interface{}{
		"agent_count":                len(rows),
		"power_agent_count":          0.0,
		"period_mode":                periodMode,
		"period_label":               periodLabel,
		"period_start":               start.Format("2006-01-02"),
		"period_end":                 end.Format("2006-01-02"),
		"period_success_money":       0.0,
		"period_success_count":       0.0,
		"period_first_commission":    0.0,
		"period_second_commission":   0.0,
		"period_commission_estimate": 0.0,
		"total_success_money":        0.0,
		"total_success_count":        0.0,
		"total_first_commission":     0.0,
		"total_second_commission":    0.0,
		"total_commission_estimate":  0.0,
		"settled_amount":             0.0,
		"pending_amount":             0.0,
	}
	for _, row := range rows {
		userID := int64(agentFloat64(row["user_id"]))
		if agentFloat64(row["power_agent"]) > 0 {
			summary["power_agent_count"] = agentFloat64(summary["power_agent_count"]) + 1
		}
		commissionRate := fallbackRate(agentFloat64(row["commission_rate"]), 0.2)
		secondEnabled := agentFloat64(row["second_commission_enabled"]) > 0
		secondRate := fallbackRate(agentFloat64(row["second_commission_rate"]), 0.03)
		periodMoney := agentFloat64(row["period_success_money"])
		totalMoney := agentFloat64(row["total_success_money"])
		periodSecondMoney := agentFloat64(row["period_second_success_money"])
		totalSecondMoney := agentFloat64(row["total_second_success_money"])
		periodFirstCommission := periodMoney * commissionRate
		totalFirstCommission := totalMoney * commissionRate
		periodSecondCommission := 0.0
		totalSecondCommission := 0.0
		if secondEnabled {
			periodSecondCommission = periodSecondMoney * secondRate
			totalSecondCommission = totalSecondMoney * secondRate
		}
		row["period_first_commission"] = periodFirstCommission
		row["period_second_commission"] = periodSecondCommission
		row["period_commission_estimate"] = periodFirstCommission + periodSecondCommission
		periodSettledAmount := periodSettledByUser[userID]
		periodPendingAmount := periodFirstCommission + periodSecondCommission - periodSettledAmount
		if periodPendingAmount < 0 {
			periodPendingAmount = 0
		}
		row["period_settled_amount"] = periodSettledAmount
		row["period_pending_amount"] = periodPendingAmount
		row["total_first_commission"] = totalFirstCommission
		row["total_second_commission"] = totalSecondCommission
		row["total_commission_estimate"] = totalFirstCommission + totalSecondCommission
		settlement := settledByUser[userID]
		settledAmount := agentFloat64(settlement["settled_amount"])
		pendingAmount := totalFirstCommission + totalSecondCommission - settledAmount
		if pendingAmount < 0 {
			pendingAmount = 0
		}
		row["settled_amount"] = settledAmount
		row["pending_amount"] = pendingAmount
		row["last_settled_at"] = settlement["last_settled_at"]
		summary["period_success_money"] = agentFloat64(summary["period_success_money"]) + periodMoney
		summary["period_success_count"] = agentFloat64(summary["period_success_count"]) + agentFloat64(row["period_success_count"])
		summary["period_first_commission"] = agentFloat64(summary["period_first_commission"]) + periodFirstCommission
		summary["period_second_commission"] = agentFloat64(summary["period_second_commission"]) + periodSecondCommission
		summary["period_commission_estimate"] = agentFloat64(summary["period_commission_estimate"]) + periodFirstCommission + periodSecondCommission
		summary["total_success_money"] = agentFloat64(summary["total_success_money"]) + totalMoney
		summary["total_success_count"] = agentFloat64(summary["total_success_count"]) + agentFloat64(row["total_success_count"])
		summary["total_first_commission"] = agentFloat64(summary["total_first_commission"]) + totalFirstCommission
		summary["total_second_commission"] = agentFloat64(summary["total_second_commission"]) + totalSecondCommission
		summary["total_commission_estimate"] = agentFloat64(summary["total_commission_estimate"]) + totalFirstCommission + totalSecondCommission
		summary["settled_amount"] = agentFloat64(summary["settled_amount"]) + settledAmount
		summary["pending_amount"] = agentFloat64(summary["pending_amount"]) + pendingAmount
	}
	return map[string]interface{}{"summary": summary, "agents": rows}, nil
}

func normalizedCommissionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "week":
		return "week"
	case "month":
		return "month"
	default:
		return "day"
	}
}

func commissionPeriod(mode, startDate, endDate string) (time.Time, time.Time, string, string) {
	if startTs, endTs, ok := parseDateRange(startDate, endDate); ok {
		start := time.Unix(startTs, 0)
		end := time.Unix(endTs, 0)
		return start, end, start.Format("2006-01-02") + " 至 " + end.Format("2006-01-02"), "custom"
	}
	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	switch normalizedCommissionMode(mode) {
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -(weekday - 1))
		return start, end, start.Format("2006-01-02") + " 至 " + end.Format("2006-01-02"), "week"
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		return start, end, start.Format("2006-01-02") + " 至 " + end.Format("2006-01-02"), "month"
	default:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		return start, end, start.Format("2006-01-02"), "day"
	}
}

func (s *AgentService) CreateCommissionSettlement(req AdminSettlementRequest) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	if req.UserID <= 0 {
		return fmt.Errorf("invalid user_id")
	}
	if req.SettledAmount <= 0 {
		return fmt.Errorf("settled_amount must be greater than 0")
	}
	startTs, endTs, ok := parseDateRange(req.PeriodStart, req.PeriodEnd)
	if !ok {
		return fmt.Errorf("invalid period range")
	}
	periodType := normalizedCommissionMode(req.PeriodType)
	if strings.TrimSpace(req.PeriodType) == "custom" {
		periodType = "custom"
	}
	now := time.Now().Unix()
	_, err := s.db.Execute(s.db.RebindQuery(`
		INSERT INTO agent_commission_settlements (user_id, period_type, period_start, period_end, settled_amount, note, settled_at, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'settled', ?)`), req.UserID, periodType, time.Unix(startTs, 0).Format("2006-01-02"), time.Unix(endTs, 0).Format("2006-01-02"), req.SettledAmount, strings.TrimSpace(req.Note), now, now)
	return err
}

func (s *AgentService) AdminSettlementRecords(mode string) ([]map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	where := "1 = 1"
	args := []interface{}{}
	if mode = normalizedSettlementMode(mode); mode != "all" {
		where = "s.period_type = ?"
		args = append(args, mode)
	}
	rows, err := s.db.Query(s.db.RebindQuery(`
		SELECT s.id, s.user_id, u.username, u.display_name, s.period_type, s.period_start, s.period_end, s.settled_amount, s.status, s.settled_at, s.note
		FROM agent_commission_settlements s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE `+where+`
		ORDER BY s.settled_at DESC, s.id DESC
		LIMIT 200`), args...)
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return rows, err
}

func (s *AgentService) AdminCommissionRedemptionRecords(limit int) ([]map[string]interface{}, error) {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(s.db.RebindQuery(`
		SELECT r.id, r.user_id, u.username, u.display_name, r.redemption_id, r.redemption_key,
			r.redemption_name, r.quota, r.commission_amount, r.status, r.created_at
		FROM agent_commission_redemptions r
		LEFT JOIN users u ON u.id = r.user_id
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT ?`), limit)
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return rows, err
}

func normalizedSettlementMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "day", "week", "month", "custom":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "all"
	}
}

func (s *AgentService) UpdateSettlementStatus(id int64, status string) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "settled" && status != "unsettled" {
		return fmt.Errorf("invalid settlement status")
	}
	if id <= 0 {
		return fmt.Errorf("invalid settlement id")
	}
	_, err := s.db.Execute(s.db.RebindQuery(`UPDATE agent_commission_settlements SET status = ? WHERE id = ?`), status, id)
	return err
}

func (s *AgentService) DeleteSettlement(id int64) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("invalid settlement id")
	}
	_, err := s.db.Execute(s.db.RebindQuery(`DELETE FROM agent_commission_settlements WHERE id = ?`), id)
	return err
}

func (s *AgentService) SaveAdminSettings(announcementHTML string, defaultRate float64, secondEnabled bool, secondRate, redemptionQuotaUnit, redemptionCommissionUnit float64) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	if defaultRate < 0 {
		defaultRate = 0
	}
	if secondRate <= 0 {
		secondRate = 0.03
	}
	if redemptionQuotaUnit <= 0 {
		redemptionQuotaUnit = 10
	}
	if redemptionCommissionUnit <= 0 {
		redemptionCommissionUnit = 3
	}
	redemptionQuotaUnit *= 500000
	enabled := 0
	if secondEnabled {
		enabled = 1
	}
	_, err := s.db.Execute(s.db.RebindQuery(`
		UPDATE agent_settings
		SET announcement_html = ?, default_commission_rate = ?, second_commission_enabled = ?, second_commission_rate = ?, redemption_quota_unit = ?, redemption_commission_unit = ?, updated_at = ?
		WHERE id = 1`), announcementHTML, defaultRate, enabled, secondRate, redemptionQuotaUnit, redemptionCommissionUnit, time.Now().Unix())
	return err
}

func (s *AgentService) SaveCommissionRate(userID int64, rate float64, secondEnabled bool, secondRate float64) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user_id")
	}
	if rate < 0 {
		rate = 0
	}
	if secondRate <= 0 {
		secondRate = 0.03
	}
	enabled := 0
	if secondEnabled {
		enabled = 1
	}
	_, err := s.db.Execute(s.db.RebindQuery(`
		INSERT INTO agent_commission_rates (user_id, commission_rate, second_commission_enabled, second_commission_rate, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE commission_rate = VALUES(commission_rate), second_commission_enabled = VALUES(second_commission_enabled), second_commission_rate = VALUES(second_commission_rate), updated_at = VALUES(updated_at)`), userID, rate, enabled, secondRate, time.Now().Unix())
	return err
}

func (s *AgentService) DeleteCommissionRate(userID int64) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	_, err := s.db.Execute(s.db.RebindQuery(`DELETE FROM agent_commission_rates WHERE user_id = ?`), userID)
	return err
}

func (s *AgentService) SetPowerAgent(userID int64, enabled bool) error {
	if err := s.EnsureAgentSettingsTables(); err != nil {
		return err
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user_id")
	}
	if enabled {
		_, err := s.db.Execute(s.db.RebindQuery(`INSERT INTO agent_power_agents (user_id, created_at) VALUES (?, ?) ON DUPLICATE KEY UPDATE created_at = VALUES(created_at)`), userID, time.Now().Unix())
		return err
	}
	_, err := s.db.Execute(s.db.RebindQuery(`DELETE FROM agent_power_agents WHERE user_id = ?`), userID)
	return err
}

func (s *AgentService) Invitees(userID int64, page, pageSize int, keyword, startDate, endDate string) (map[string]interface{}, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	where := []string{"u.inviter_id = ?", "u.deleted_at IS NULL"}
	args := []interface{}{userID}
	startTs, endTs, hasRange := parseDateRange(startDate, endDate)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where = append(where, "(u.username LIKE ? OR COALESCE(u.display_name, '') LIKE ? OR COALESCE(u.email, '') LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	whereSQL := strings.Join(where, " AND ")
	bucket := topUpBucketSQL("t.status")
	secondEnabled, secondRate := s.secondCommissionSettings(userID)
	secondRateSQL := "0"
	if secondEnabled {
		secondRateSQL = fmt.Sprintf("%f", secondRate)
	}
	countQuery := s.db.RebindQuery(fmt.Sprintf("SELECT COUNT(*) FROM users u WHERE %s", whereSQL))
	var total int64
	if err := s.db.DB.Get(&total, countQuery, args...); err != nil {
		return nil, err
	}
	selectArgs := make([]interface{}, 0, len(args)+4)
	topupJoinFilter := ""
	if hasRange {
		topupJoinFilter = " AND t.create_time >= ? AND t.create_time <= ?"
		selectArgs = append(selectArgs, startTs, endTs)
	}
	selectArgs = append(selectArgs, args...)
	selectQuery := s.db.RebindQuery(fmt.Sprintf(`
		SELECT u.id, u.username, u.display_name, u.email, u.status, u.quota, u.used_quota,
			u.request_count, u.created_at, u.last_login_at,
			COUNT(t.id) AS topup_count,
			COALESCE(SUM(CASE WHEN (%s) = 'success' THEN t.money ELSE 0 END), 0) AS success_money,
			MAX(t.complete_time) AS last_topup_at,
			COALESCE(MAX(sub.sub_invitees), 0) AS sub_invitees,
			COALESCE(MAX(sub.sub_qualified_invitees), 0) AS sub_qualified_invitees,
			COALESCE(MAX(sub.sub_success_money), 0) AS sub_success_money,
			COALESCE(MAX(r.commission_rate), MAX(settings.default_commission_rate)) AS commission_rate,
			%s AS second_commission_rate,
			COALESCE(MAX(sub.sub_success_money), 0) * %s AS second_commission_estimate
		FROM users u
		LEFT JOIN top_ups t ON t.user_id = u.id%s
		LEFT JOIN agent_settings settings ON settings.id = 1
		LEFT JOIN agent_commission_rates r ON r.user_id = u.id
		LEFT JOIN (
			SELECT child.inviter_id,
				COUNT(*) AS sub_invitees,
				SUM(CASE WHEN COALESCE(pay.success_money, 0) > 10 THEN 1 ELSE 0 END) AS sub_qualified_invitees,
				COALESCE(SUM(pay.success_money), 0) AS sub_success_money
			FROM users child
			LEFT JOIN (
				SELECT user_id, SUM(CASE WHEN LOWER(TRIM(COALESCE(pay_t.status, ''))) = 'success' THEN money ELSE 0 END) AS success_money
				FROM top_ups pay_t
				GROUP BY user_id
			) pay ON pay.user_id = child.id
			WHERE child.deleted_at IS NULL
			GROUP BY child.inviter_id
		) sub ON sub.inviter_id = u.id
		WHERE %s
		GROUP BY u.id
		ORDER BY u.id DESC
		LIMIT ? OFFSET ?`, bucket, secondRateSQL, secondRateSQL, topupJoinFilter, whereSQL))
	selectArgs = append(selectArgs, pageSize, offset)
	var items []AgentInvitee
	if err := s.db.DB.Select(&items, selectQuery, selectArgs...); err != nil {
		return nil, err
	}
	if items == nil {
		items = []AgentInvitee{}
	}
	return paginated(items, total, page, pageSize), nil
}

func (s *AgentService) InviteeRanking(userID int64, days, limit int) ([]AgentRankUser, error) {
	if days <= 0 {
		days = 30
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	start := time.Now().AddDate(0, 0, -(days - 1))
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	query := s.db.RebindQuery(`
		SELECT u.id AS user_id, u.username, u.display_name,
			COUNT(t.id) AS success_count,
			COALESCE(SUM(t.money), 0) AS success_money,
			COALESCE(MAX(sub.sub_invitees), 0) AS sub_invitees,
			COALESCE(MAX(sub.sub_success_money), 0) AS sub_success_money
		FROM users u
		JOIN top_ups t ON t.user_id = u.id AND LOWER(TRIM(COALESCE(t.status, ''))) = 'success' AND t.create_time >= ?
		LEFT JOIN (
			SELECT child.inviter_id, COUNT(*) AS sub_invitees, COALESCE(SUM(pay.success_money), 0) AS sub_success_money
			FROM users child
			LEFT JOIN (
				SELECT user_id, SUM(CASE WHEN LOWER(TRIM(COALESCE(status, ''))) = 'success' THEN money ELSE 0 END) AS success_money
				FROM top_ups
				GROUP BY user_id
			) pay ON pay.user_id = child.id
			WHERE child.deleted_at IS NULL
			GROUP BY child.inviter_id
		) sub ON sub.inviter_id = u.id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL
		GROUP BY u.id, u.username, u.display_name
		ORDER BY success_money DESC
		LIMIT ?`)
	var rows []AgentRankUser
	if err := s.db.DB.Select(&rows, query, start.Unix(), userID, limit); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AgentRankUser{}
	}
	return rows, nil
}

func (s *AgentService) InviteeTrend(userID int64, mode string) ([]map[string]interface{}, error) {
	bucket := "LOWER(TRIM(COALESCE(t.status, '')))"
	now := time.Now()
	var start time.Time
	count := 10
	fillMode := "day"
	switch mode {
	case "week":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		count = 5
		fillMode = "week"
	case "month":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)
		count = 12
		fillMode = "month"
	default:
		start = now.AddDate(0, 0, -9)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	}
	rows, err := s.db.Query(s.db.RebindQuery(fmt.Sprintf(`
		SELECT t.create_time, t.money
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE u.inviter_id = ? AND u.deleted_at IS NULL AND %s = 'success' AND t.create_time >= ?`, bucket)), userID, start.Unix())
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return aggregateTrendMoneyRows(rows, start, count, fillMode), nil
}

func (s *AgentService) TopUps(userID int64, page, pageSize int, status, keyword, startDate, endDate string) (map[string]interface{}, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	bucket := topUpBucketSQL("t.status")
	where := []string{"u.inviter_id = ?", "u.deleted_at IS NULL"}
	args := []interface{}{userID}
	startTs, endTs, hasRange := parseDateRange(startDate, endDate)
	if hasRange {
		where = append(where, "t.create_time >= ? AND t.create_time <= ?")
		args = append(args, startTs, endTs)
	}
	if status != "" {
		allowed := map[string]bool{"success": true, "pending": true, "failed": true, "expired": true, "unknown": true}
		if allowed[status] {
			where = append(where, fmt.Sprintf("%s = ?", bucket))
			args = append(args, status)
		}
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where = append(where, "(u.username LIKE ? OR COALESCE(u.display_name, '') LIKE ? OR COALESCE(t.trade_no, '') LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	whereSQL := strings.Join(where, " AND ")
	countQuery := s.db.RebindQuery(fmt.Sprintf("SELECT COUNT(*) FROM top_ups t JOIN users u ON u.id = t.user_id WHERE %s", whereSQL))
	var total int64
	if err := s.db.DB.Get(&total, countQuery, args...); err != nil {
		return nil, err
	}
	selectQuery := s.db.RebindQuery(fmt.Sprintf(`
		SELECT t.id, t.user_id, u.username, u.display_name, t.money, t.amount,
			t.trade_no, t.payment_method, %s AS payment_provider, t.create_time,
			t.complete_time, COALESCE(t.status, '') AS raw_status, %s AS status_bucket
		FROM top_ups t
		JOIN users u ON u.id = t.user_id
		WHERE %s
		ORDER BY COALESCE(t.complete_time, t.create_time) DESC, t.id DESC
		LIMIT ? OFFSET ?`, topUpPaymentProviderExpr("t"), bucket, whereSQL))
	args = append(args, pageSize, offset)
	var items []AgentTopUp
	if err := s.db.DB.Select(&items, selectQuery, args...); err != nil {
		return nil, err
	}
	if items == nil {
		items = []AgentTopUp{}
	}
	return paginated(items, total, page, pageSize), nil
}

func ParseAgentSubject(subject string) (int64, bool) {
	if !strings.HasPrefix(subject, "agent:") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(subject, "agent:"), 10, 64)
	return id, err == nil && id > 0
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func parseDateRange(startDate, endDate string) (int64, int64, bool) {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	if startDate == "" || endDate == "" {
		return 0, 0, false
	}
	loc := time.Local
	start, err := time.ParseInLocation("2006-01-02", startDate, loc)
	if err != nil {
		return 0, 0, false
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, loc)
	if err != nil {
		return 0, 0, false
	}
	end = end.Add(24*time.Hour - time.Second)
	if end.Before(start) {
		return 0, 0, false
	}
	return start.Unix(), end.Unix(), true
}

func dateSQL(column string, hasRange bool) string {
	if !hasRange {
		return ""
	}
	return fmt.Sprintf(" AND %s >= ? AND %s <= ?", column, column)
}

func dateArgs(userID, startTs, endTs int64, hasRange bool) []interface{} {
	args := []interface{}{userID}
	if hasRange {
		args = append(args, startTs, endTs)
	}
	return args
}

func firstRow(rows []map[string]interface{}) map[string]interface{} {
	if len(rows) == 0 || rows[0] == nil {
		return map[string]interface{}{}
	}
	return rows[0]
}

func agentFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case nil:
		return 0
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case []byte:
		f, _ := strconv.ParseFloat(string(val), 64)
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func agentRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func fallbackRate(value, fallback float64) float64 {
	if value <= 0 {
		return fallback
	}
	return value
}

func decodeAgentAffCode(value interface{}) string {
	var text string
	switch v := value.(type) {
	case nil:
		return ""
	case []byte:
		text = string(v)
	case string:
		text = v
	default:
		text = fmt.Sprint(v)
	}
	text = strings.TrimSpace(text)
	if text == "" || text == "<nil>" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err == nil && decoded != nil && utf8.Valid(decoded) {
		plain := strings.TrimSpace(string(decoded))
		if plain != "" && isPrintableAgentCode(plain) {
			return plain
		}
	}
	return text
}

func isPrintableAgentCode(text string) bool {
	for _, r := range text {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

const agentInviteFallbackBaseURL = "https://newapi.omgteam.me"

func buildAgentInviteURL(aff interface{}) string {
	code := decodeAgentAffCode(aff)
	if code == "" {
		return ""
	}
	cfg := config.Get()
	baseURL := strings.TrimSpace(cfg.NewAPIPublicBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(cfg.NewAPIBaseURL)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" || isLoopbackAgentBaseURL(baseURL) {
		baseURL = agentInviteFallbackBaseURL
	}
	return baseURL + "/register?aff=" + code
}

func isLoopbackAgentBaseURL(baseURL string) bool {
	lower := strings.ToLower(strings.TrimSpace(baseURL))
	return strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "0.0.0.0")
}

func fillDailyRows(rows []map[string]interface{}, start time.Time, days int) []map[string]interface{} {
	byDate := map[string]map[string]interface{}{}
	for _, row := range rows {
		key := fmt.Sprint(row["date"])
		byDate[key] = row
	}
	result := make([]map[string]interface{}, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		if row, ok := byDate[date]; ok {
			result = append(result, row)
		} else {
			result = append(result, map[string]interface{}{"date": date, "success_count": 0, "success_money": 0})
		}
	}
	return result
}

func aggregateDailyMoneyRows(rows []map[string]interface{}, start time.Time, days int) []map[string]interface{} {
	byDate := map[string]map[string]interface{}{}
	for _, row := range rows {
		createdAt := int64(agentFloat64(row["create_time"]))
		date := time.Unix(createdAt, 0).In(time.Local).Format("2006-01-02")
		current := byDate[date]
		if current == nil {
			current = map[string]interface{}{"date": date, "success_count": 0.0, "success_money": 0.0}
			byDate[date] = current
		}
		current["success_count"] = agentFloat64(current["success_count"]) + 1
		current["success_money"] = agentFloat64(current["success_money"]) + agentFloat64(row["money"])
	}
	result := make([]map[string]interface{}, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		if row := byDate[date]; row != nil {
			result = append(result, row)
		} else {
			result = append(result, map[string]interface{}{"date": date, "success_count": 0, "success_money": 0})
		}
	}
	return result
}

func aggregateCommissionMoneyRows(rows []map[string]interface{}, start time.Time, count int, mode string, commissionRate float64) []map[string]interface{} {
	byLabel := map[string]map[string]interface{}{}
	for _, row := range rows {
		createdAt := int64(agentFloat64(row["create_time"]))
		date := time.Unix(createdAt, 0).In(time.Local)
		label := date.Format("2006-01-02")
		if mode == "week" {
			label = fmt.Sprintf("%d", yearWeek(date))
		} else if mode == "month" {
			label = date.Format("2006-01")
		}
		current := byLabel[label]
		if current == nil {
			current = map[string]interface{}{"label": label, "commission_amount": 0.0}
			byLabel[label] = current
		}
		current["commission_amount"] = agentFloat64(current["commission_amount"]) + agentFloat64(row["money"])*commissionRate
	}
	result := make([]map[string]interface{}, 0, count)
	for i := 0; i < count; i++ {
		label := start.AddDate(0, 0, i).Format("2006-01-02")
		if mode == "week" {
			label = fmt.Sprintf("%d", yearWeek(start.AddDate(0, 0, i*7)))
		} else if mode == "month" {
			label = start.AddDate(0, i, 0).Format("2006-01")
		}
		if row := byLabel[label]; row != nil {
			result = append(result, row)
		} else {
			result = append(result, map[string]interface{}{"label": label, "commission_amount": 0})
		}
	}
	return result
}

func aggregateTrendMoneyRows(rows []map[string]interface{}, start time.Time, count int, mode string) []map[string]interface{} {
	byLabel := map[string]map[string]interface{}{}
	for _, row := range rows {
		createdAt := int64(agentFloat64(row["create_time"]))
		date := time.Unix(createdAt, 0).In(time.Local)
		label := date.Format("2006-01-02")
		if mode == "week" {
			label = fmt.Sprintf("%d", yearWeek(date))
		} else if mode == "month" {
			label = date.Format("2006-01")
		}
		current := byLabel[label]
		if current == nil {
			current = map[string]interface{}{"label": label, "success_money": 0.0, "success_count": 0.0}
			byLabel[label] = current
		}
		current["success_money"] = agentFloat64(current["success_money"]) + agentFloat64(row["money"])
		current["success_count"] = agentFloat64(current["success_count"]) + 1
	}
	result := make([]map[string]interface{}, 0, count)
	for i := 0; i < count; i++ {
		label := start.AddDate(0, 0, i).Format("2006-01-02")
		if mode == "week" {
			label = fmt.Sprintf("%d", yearWeek(start.AddDate(0, 0, i*7)))
		} else if mode == "month" {
			label = start.AddDate(0, i, 0).Format("2006-01")
		}
		if row := byLabel[label]; row != nil {
			result = append(result, row)
		} else {
			result = append(result, map[string]interface{}{"label": label, "success_count": 0, "success_money": 0})
		}
	}
	return result
}

func fillTrendRows(rows []map[string]interface{}, start time.Time, count int, mode string) []map[string]interface{} {
	byLabel := map[string]map[string]interface{}{}
	for _, row := range rows {
		byLabel[fmt.Sprint(row["label"])] = row
	}
	result := make([]map[string]interface{}, 0, count)
	for i := 0; i < count; i++ {
		var label string
		if mode == "week" {
			label = fmt.Sprintf("%d", yearWeek(start.AddDate(0, 0, i*7)))
		} else {
			label = start.AddDate(0, 0, i).Format("2006-01-02")
		}
		if row, ok := byLabel[label]; ok {
			result = append(result, row)
		} else {
			result = append(result, map[string]interface{}{"label": label, "success_count": 0, "success_money": 0})
		}
	}
	return result
}

func mergeCommissionTrendRows(commissionRows, settledRows []map[string]interface{}) []map[string]interface{} {
	merged := map[string]map[string]interface{}{}
	for _, row := range commissionRows {
		label := fmt.Sprint(row["label"])
		merged[label] = map[string]interface{}{"label": label, "commission_amount": agentFloat64(row["commission_amount"]), "settled_amount": 0.0}
	}
	for _, row := range settledRows {
		label := fmt.Sprint(row["label"])
		if _, ok := merged[label]; !ok {
			merged[label] = map[string]interface{}{"label": label, "commission_amount": 0.0, "settled_amount": 0.0}
		}
		merged[label]["settled_amount"] = agentFloat64(row["settled_amount"])
	}
	result := make([]map[string]interface{}, 0, len(merged))
	for _, row := range merged {
		commission := agentFloat64(row["commission_amount"])
		settled := agentFloat64(row["settled_amount"])
		pending := commission - settled
		if pending < 0 {
			pending = 0
		}
		row["pending_amount"] = pending
		result = append(result, row)
	}
	return result
}

func fillCommissionTrendRows(rows []map[string]interface{}, start time.Time, count int, mode string) []map[string]interface{} {
	byLabel := map[string]map[string]interface{}{}
	for _, row := range rows {
		byLabel[fmt.Sprint(row["label"])] = row
	}
	result := make([]map[string]interface{}, 0, count)
	for i := 0; i < count; i++ {
		var label string
		if mode == "week" {
			label = fmt.Sprintf("%d", yearWeek(start.AddDate(0, 0, i*7)))
		} else if mode == "month" {
			label = start.AddDate(0, i, 0).Format("2006-01")
		} else {
			label = start.AddDate(0, 0, i).Format("2006-01-02")
		}
		row, ok := byLabel[label]
		if !ok {
			row = map[string]interface{}{"label": label, "commission_amount": 0, "settled_amount": 0, "pending_amount": 0}
		}
		result = append(result, row)
	}
	return result
}

func applySettledCoverage(rows []map[string]interface{}, ranges []map[string]interface{}, mode string) {
	for _, row := range rows {
		label := fmt.Sprint(row["label"])
		commission := agentFloat64(row["commission_amount"])
		covered := false
		for _, item := range ranges {
			start, end, ok := parseSettlementDateRange(fmt.Sprint(item["period_start"]), fmt.Sprint(item["period_end"]))
			if !ok {
				continue
			}
			if trendLabelCovered(label, mode, start, end) {
				covered = true
				break
			}
		}
		if covered {
			row["settled_amount"] = commission
			row["pending_amount"] = 0.0
		} else {
			row["settled_amount"] = 0.0
			row["pending_amount"] = commission
		}
	}
}

func parseSettlementDateRange(startDate, endDate string) (time.Time, time.Time, bool) {
	start, err1 := time.ParseInLocation("2006-01-02", startDate, time.Local)
	end, err2 := time.ParseInLocation("2006-01-02", endDate, time.Local)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, false
	}
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
	return start, end, true
}

func trendLabelCovered(label, mode string, start, end time.Time) bool {
	if mode == "month" {
		current, err := time.ParseInLocation("2006-01", label, time.Local)
		if err != nil {
			return false
		}
		monthEnd := current.AddDate(0, 1, 0).Add(-time.Second)
		return !monthEnd.Before(start) && !current.After(end)
	}
	if mode == "week" {
		weekInt, err := strconv.Atoi(label)
		if err != nil {
			return false
		}
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			if yearWeek(day) == weekInt {
				return true
			}
		}
		return false
	}
	current, err := time.ParseInLocation("2006-01-02", label, time.Local)
	if err != nil {
		return false
	}
	return !current.Before(start) && !current.After(end)
}

func yearWeek(t time.Time) int {
	y, w := t.ISOWeek()
	return y*100 + w
}

func paginated(items interface{}, total int64, page, pageSize int) map[string]interface{} {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	return map[string]interface{}{
		"items":       items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	}
}
