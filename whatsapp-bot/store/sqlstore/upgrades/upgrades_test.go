package upgrades_test

import (
	"testing"

	"zaelix/store/sqlstore/upgrades"
)

func TestTable_Registered(t *testing.T) {
	if len(upgrades.Table) == 0 {
		t.Error("upgrades.Table has no registered upgrades; expected at least one")
	}
}
