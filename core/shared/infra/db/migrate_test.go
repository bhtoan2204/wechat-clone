package db

import (
	"strings"
	"testing"
)

func TestSplitSQLStatements_OracleTriggerBlocks(t *testing.T) {
	input := `
-- regular table
CREATE TABLE ledger_transactions (
    transaction_id VARCHAR2(1024) PRIMARY KEY
);

CREATE OR REPLACE TRIGGER trg_ledger_entries_append_only
BEFORE UPDATE OR DELETE ON ledger_entries
FOR EACH ROW
BEGIN
    raise_application_error(-20001, 'ledger_entries is append-only');
END;
/

CREATE INDEX idx_ledger_entries_transaction_id ON ledger_entries(transaction_id);
`

	statements := splitSQLStatements(input)
	if len(statements) != 3 {
		t.Fatalf("expected 3 statements, got %d: %#v", len(statements), statements)
	}

	if !strings.Contains(statements[1], "raise_application_error") {
		t.Fatalf("expected trigger block to stay intact, got %q", statements[1])
	}

	if strings.Contains(statements[1], "\n/") || strings.HasSuffix(strings.TrimSpace(statements[1]), "/") {
		t.Fatalf("expected sqlplus terminator to be removed from trigger block: %q", statements[1])
	}
}
