package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/kudaompq/ai_trending/backend/internal/database"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// OpportunityRepository handles opportunity persistence
type OpportunityRepository struct {
	db *sql.DB
}

// NewOpportunityRepository creates a new opportunity repository
func NewOpportunityRepository() *OpportunityRepository {
	return &OpportunityRepository{
		db: database.DB,
	}
}

// Save saves an opportunity to the database
func (r *OpportunityRepository) Save(opp *model.TradingOpportunity) error {
	// Serialize complex fields to JSON
	entryReasons, _ := json.Marshal(opp.Entry.Reasons)
	takeProfitLevels, _ := json.Marshal(opp.TakeProfit)
	confidenceFactors, _ := json.Marshal(opp.Confidence.Factors)

	now := time.Now().Unix() * 1000

	query := `
		INSERT OR REPLACE INTO opportunities (
			id, symbol, type, strategy, timestamp,
			entry_price, entry_reasons,
			stop_loss_price, stop_loss_distance_pct, stop_loss_method,
			take_profit_levels,
			risk_reward_ratio, risk_amount, reward_amount, risk_pct, reward_pct,
			confidence_score, confidence_level, confidence_factors,
			expires_at, status,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		opp.ID, opp.Symbol, opp.Type, opp.Strategy, opp.Timestamp,
		opp.Entry.Price, string(entryReasons),
		opp.StopLoss.Price, opp.StopLoss.DistancePct, opp.StopLoss.Method,
		string(takeProfitLevels),
		opp.RiskReward.Ratio, opp.RiskReward.RiskAmount, opp.RiskReward.RewardAmount,
		opp.RiskReward.RiskPct, opp.RiskReward.RewardPct,
		opp.Confidence.Score, opp.Confidence.Level, string(confidenceFactors),
		opp.Validity.ExpiresAt, opp.Validity.Status,
		now, now,
	)

	return err
}

// FindByID finds an opportunity by ID
func (r *OpportunityRepository) FindByID(id string) (*model.TradingOpportunity, error) {
	query := `
		SELECT id, symbol, type, strategy, timestamp,
			entry_price, entry_reasons,
			stop_loss_price, stop_loss_distance_pct, stop_loss_method,
			take_profit_levels,
			risk_reward_ratio, risk_amount, reward_amount, risk_pct, reward_pct,
			confidence_score, confidence_level, confidence_factors,
			expires_at, status
		FROM opportunities
		WHERE id = ?
	`

	var opp model.TradingOpportunity
	var entryReasons, takeProfitLevels, confidenceFactors string

	err := r.db.QueryRow(query, id).Scan(
		&opp.ID, &opp.Symbol, &opp.Type, &opp.Strategy, &opp.Timestamp,
		&opp.Entry.Price, &entryReasons,
		&opp.StopLoss.Price, &opp.StopLoss.DistancePct, &opp.StopLoss.Method,
		&takeProfitLevels,
		&opp.RiskReward.Ratio, &opp.RiskReward.RiskAmount, &opp.RiskReward.RewardAmount,
		&opp.RiskReward.RiskPct, &opp.RiskReward.RewardPct,
		&opp.Confidence.Score, &opp.Confidence.Level, &confidenceFactors,
		&opp.Validity.ExpiresAt, &opp.Validity.Status,
	)

	if err != nil {
		return nil, err
	}

	// Deserialize JSON fields
	json.Unmarshal([]byte(entryReasons), &opp.Entry.Reasons)
	json.Unmarshal([]byte(takeProfitLevels), &opp.TakeProfit)
	json.Unmarshal([]byte(confidenceFactors), &opp.Confidence.Factors)

	return &opp, nil
}

// FindBySymbol finds opportunities by symbol and status
func (r *OpportunityRepository) FindBySymbol(symbol string, status string) ([]model.TradingOpportunity, error) {
	query := `
		SELECT id, symbol, type, strategy, timestamp,
			entry_price, entry_reasons,
			stop_loss_price, stop_loss_distance_pct, stop_loss_method,
			take_profit_levels,
			risk_reward_ratio, risk_amount, reward_amount, risk_pct, reward_pct,
			confidence_score, confidence_level, confidence_factors,
			expires_at, status
		FROM opportunities
		WHERE symbol = ? AND status = ?
		ORDER BY timestamp DESC
	`

	rows, err := r.db.Query(query, symbol, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanOpportunities(rows)
}

// FindActive finds all active opportunities
func (r *OpportunityRepository) FindActive() ([]model.TradingOpportunity, error) {
	query := `
		SELECT id, symbol, type, strategy, timestamp,
			entry_price, entry_reasons,
			stop_loss_price, stop_loss_distance_pct, stop_loss_method,
			take_profit_levels,
			risk_reward_ratio, risk_amount, reward_amount, risk_pct, reward_pct,
			confidence_score, confidence_level, confidence_factors,
			expires_at, status
		FROM opportunities
		WHERE status = 'ACTIVE'
		ORDER BY timestamp DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanOpportunities(rows)
}

// UpdateStatus updates the status of an opportunity
func (r *OpportunityRepository) UpdateStatus(id string, status string) error {
	query := `UPDATE opportunities SET status = ?, updated_at = ? WHERE id = ?`
	now := time.Now().Unix() * 1000
	_, err := r.db.Exec(query, status, now, id)
	return err
}

// UpdateExpiredOpportunities marks expired opportunities as EXPIRED
func (r *OpportunityRepository) UpdateExpiredOpportunities() error {
	now := time.Now().Unix() * 1000
	query := `UPDATE opportunities SET status = 'EXPIRED', updated_at = ? WHERE expires_at < ? AND status = 'ACTIVE'`
	_, err := r.db.Exec(query, now, now)
	return err
}

// DeleteOld deletes opportunities older than the specified days
func (r *OpportunityRepository) DeleteOld(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days).Unix() * 1000
	query := `DELETE FROM opportunities WHERE timestamp < ?`
	_, err := r.db.Exec(query, cutoff)
	return err
}

// GetHistory gets historical opportunities for a symbol
func (r *OpportunityRepository) GetHistory(symbol string, limit int) ([]model.TradingOpportunity, error) {
	query := `
		SELECT id, symbol, type, strategy, timestamp,
			entry_price, entry_reasons,
			stop_loss_price, stop_loss_distance_pct, stop_loss_method,
			take_profit_levels,
			risk_reward_ratio, risk_amount, reward_amount, risk_pct, reward_pct,
			confidence_score, confidence_level, confidence_factors,
			expires_at, status
		FROM opportunities
		WHERE symbol = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanOpportunities(rows)
}

// scanOpportunities scans multiple opportunities from rows
func (r *OpportunityRepository) scanOpportunities(rows *sql.Rows) ([]model.TradingOpportunity, error) {
	opportunities := []model.TradingOpportunity{}

	for rows.Next() {
		var opp model.TradingOpportunity
		var entryReasons, takeProfitLevels, confidenceFactors string

		err := rows.Scan(
			&opp.ID, &opp.Symbol, &opp.Type, &opp.Strategy, &opp.Timestamp,
			&opp.Entry.Price, &entryReasons,
			&opp.StopLoss.Price, &opp.StopLoss.DistancePct, &opp.StopLoss.Method,
			&takeProfitLevels,
			&opp.RiskReward.Ratio, &opp.RiskReward.RiskAmount, &opp.RiskReward.RewardAmount,
			&opp.RiskReward.RiskPct, &opp.RiskReward.RewardPct,
			&opp.Confidence.Score, &opp.Confidence.Level, &confidenceFactors,
			&opp.Validity.ExpiresAt, &opp.Validity.Status,
		)

		if err != nil {
			return nil, err
		}

		// Deserialize JSON fields
		json.Unmarshal([]byte(entryReasons), &opp.Entry.Reasons)
		json.Unmarshal([]byte(takeProfitLevels), &opp.TakeProfit)
		json.Unmarshal([]byte(confidenceFactors), &opp.Confidence.Factors)

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}
