package database

import (
	"time"
)

// BackfillService handles backfill-related database operations
type BackfillService struct {
	db *V2DB
}

// NewBackfillService creates a new backfill service
func NewBackfillService(db *V2DB) *BackfillService {
	return &BackfillService{db: db}
}

// InsertBackfillRecord inserts a backfill record as a transaction
func (bs *BackfillService) InsertBackfillRecord(record *BackfillRecord) error {
	tx := &Transaction{
		FromUser:        record.FromUser,
		ToUser:          record.ToUser,
		Points:          record.Points,
		Reason:          record.Reason,
		TransactionType: record.TransactionType,
		EmojiName:       record.EmojiName,
		ChannelID:       &record.ChannelID,
		ChannelName:     nil, // Channel name resolution not implemented in backfill
		MessageID:       &record.MessageID,
		Timestamp:       record.Timestamp,
		Year:            record.Timestamp.Year(),
	}

	transactionService := NewTransactionService(bs.db)
	return transactionService.InsertTransaction(tx)
}

// InsertBackfillRecords inserts multiple backfill records efficiently using bulk insert
func (bs *BackfillService) InsertBackfillRecords(records []*BackfillRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Convert backfill records to transactions
	transactions := make([]*Transaction, len(records))
	for i, record := range records {
		transactions[i] = &Transaction{
			FromUser:        record.FromUser,
			ToUser:          record.ToUser,
			Points:          record.Points,
			Reason:          record.Reason,
			TransactionType: record.TransactionType,
			EmojiName:       record.EmojiName,
			ChannelID:       &record.ChannelID,
			ChannelName:     nil, // Channel name resolution not implemented in backfill
			MessageID:       &record.MessageID,
			Timestamp:       record.Timestamp,
			Year:            record.Timestamp.Year(),
		}
	}

	transactionService := NewTransactionService(bs.db)
	return transactionService.InsertTransactionsBulk(transactions)
}

// GetBackfillStats returns statistics for backfill operations in a time range
func (bs *BackfillService) GetBackfillStats(startTime, endTime time.Time) (*BackfillStats, error) {
	query := `
		SELECT
			COUNT(*) as total_records,
			COUNT(CASE WHEN reason = 'backfill' THEN 1 END) as backfill_records,
			COUNT(CASE WHEN transaction_type = 'reactji' THEN 1 END) as emoji_records
		FROM karma_transactions
		WHERE timestamp BETWEEN $1 AND $2
	`

	var totalRecords, backfillRecords, emojiRecords int
	err := bs.db.SQL.QueryRow(query, startTime, endTime).Scan(&totalRecords, &backfillRecords, &emojiRecords)
	if err != nil {
		return nil, err
	}

	stats := &BackfillStats{
		MessagesProcessed: totalRecords,
		KarmaFound:        backfillRecords + emojiRecords,
		RecordsAdded:      backfillRecords,
		DuplicatesSkipped: 0, // This would need to be tracked during backfill
		ErrorsEncountered: 0, // This would need to be tracked during backfill
		DurationMs:        int(endTime.Sub(startTime).Milliseconds()),
	}

	return stats, nil
}
