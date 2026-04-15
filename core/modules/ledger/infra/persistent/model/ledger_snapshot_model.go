package model

import "time"

type LedgerSnapshotModel struct {
	ID            string    `gorm:"primaryKey"`
	AggregateID   string    `gorm:"not null;index:idx_ledger_snapshots_agg_ver"`
	AggregateType string    `gorm:"not null;index:idx_ledger_snapshots_agg_ver"`
	Version       int       `gorm:"not null;index:idx_ledger_snapshots_agg_ver"`
	SnapshotData  string    `gorm:"type:CLOB;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (LedgerSnapshotModel) TableName() string {
	return "ledger_snapshots"
}
