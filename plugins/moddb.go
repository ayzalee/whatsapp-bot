package plugins

import (
	"strings"
	"time"
)

func initModTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS warns (
			chat_jid TEXT,
			user_id  TEXT,
			count    INTEGER DEFAULT 0,
			PRIMARY KEY (chat_jid, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS shh_users (
			chat_jid TEXT,
			user_id  TEXT,
			PRIMARY KEY (chat_jid, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS antilink_settings (
			chat_jid TEXT PRIMARY KEY,
			mode     TEXT DEFAULT 'off'
		)`,
		`CREATE TABLE IF NOT EXISTS antiword_settings (
			chat_jid TEXT,
			word     TEXT,
			PRIMARY KEY (chat_jid, word)
		)`,
		`CREATE TABLE IF NOT EXISTS antispam_settings (
			chat_jid TEXT PRIMARY KEY,
			mode     TEXT DEFAULT 'off'
		)`,
		`CREATE TABLE IF NOT EXISTS antispam_whitelist (
			chat_jid TEXT,
			user_id  TEXT,
			PRIMARY KEY (chat_jid, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS afk_status (
			user_id TEXT PRIMARY KEY,
			message TEXT,
			set_at  INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS filters (
			scope    TEXT,
			chat_jid TEXT,
			keyword  TEXT,
			response TEXT,
			PRIMARY KEY (scope, chat_jid, keyword)
		)`,
		`CREATE TABLE IF NOT EXISTS antistatus_settings (
			chat_jid TEXT PRIMARY KEY,
			enabled  INTEGER DEFAULT 0
		)`,
	}
	for _, q := range tables {
		if _, err := settingsDB.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// ── warns ─────────────────────────────────────────────────────────────────────

func addWarn(chatJID, userID string) int {
	settingsDB.Exec(
		`INSERT INTO warns (chat_jid, user_id, count) VALUES (?, ?, 1)
		 ON CONFLICT(chat_jid, user_id) DO UPDATE SET count = count + 1`,
		chatJID, userID,
	)
	return getWarnCount(chatJID, userID)
}

func resetWarns(chatJID, userID string) {
	settingsDB.Exec(`DELETE FROM warns WHERE chat_jid = ? AND user_id = ?`, chatJID, userID)
}

func getWarnCount(chatJID, userID string) int {
	var n int
	settingsDB.QueryRow(
		`SELECT count FROM warns WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	).Scan(&n)
	return n
}

// ── shh ───────────────────────────────────────────────────────────────────────

func isShhed(chatJID, userID string) bool {
	var dummy string
	err := settingsDB.QueryRow(
		`SELECT user_id FROM shh_users WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	).Scan(&dummy)
	return err == nil
}

func setShh(chatJID, userID string) {
	settingsDB.Exec(
		`INSERT OR IGNORE INTO shh_users (chat_jid, user_id) VALUES (?, ?)`, chatJID, userID,
	)
}

func setUnShh(chatJID, userID string) {
	settingsDB.Exec(
		`DELETE FROM shh_users WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	)
}

// ── antilink ──────────────────────────────────────────────────────────────────

func getAntilinkMode(chatJID string) string {
	var mode string
	if err := settingsDB.QueryRow(
		`SELECT mode FROM antilink_settings WHERE chat_jid = ?`, chatJID,
	).Scan(&mode); err != nil {
		return "off"
	}
	return mode
}

func setAntilinkMode(chatJID, mode string) {
	settingsDB.Exec(
		`INSERT INTO antilink_settings (chat_jid, mode) VALUES (?, ?)
		 ON CONFLICT(chat_jid) DO UPDATE SET mode = excluded.mode`,
		chatJID, mode,
	)
}

// ── antiword ──────────────────────────────────────────────────────────────────

func getAntiwords(chatJID string) []string {
	rows, err := settingsDB.Query(
		`SELECT word FROM antiword_settings WHERE chat_jid = ?`, chatJID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var words []string
	for rows.Next() {
		var w string
		if rows.Scan(&w) == nil {
			words = append(words, w)
		}
	}
	return words
}

func addAntiword(chatJID, word string) {
	settingsDB.Exec(
		`INSERT OR IGNORE INTO antiword_settings (chat_jid, word) VALUES (?, ?)`, chatJID, word,
	)
}

func removeAntiword(chatJID, word string) {
	settingsDB.Exec(
		`DELETE FROM antiword_settings WHERE chat_jid = ? AND word = ?`, chatJID, word,
	)
}

// ── antispam ──────────────────────────────────────────────────────────────────

func getAntispamMode(chatJID string) string {
	var mode string
	if err := settingsDB.QueryRow(
		`SELECT mode FROM antispam_settings WHERE chat_jid = ?`, chatJID,
	).Scan(&mode); err != nil {
		return "off"
	}
	return mode
}

func setAntispamMode(chatJID, mode string) {
	settingsDB.Exec(
		`INSERT INTO antispam_settings (chat_jid, mode) VALUES (?, ?)
		 ON CONFLICT(chat_jid) DO UPDATE SET mode = excluded.mode`,
		chatJID, mode,
	)
}

func isAntispamWhitelisted(chatJID, userID string) bool {
	var dummy string
	err := settingsDB.QueryRow(
		`SELECT user_id FROM antispam_whitelist WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
	).Scan(&dummy)
	return err == nil
}

func setAntispamWhitelist(chatJID, userID string, allow bool) {
	if allow {
		settingsDB.Exec(
			`INSERT OR IGNORE INTO antispam_whitelist (chat_jid, user_id) VALUES (?, ?)`, chatJID, userID,
		)
	} else {
		settingsDB.Exec(
			`DELETE FROM antispam_whitelist WHERE chat_jid = ? AND user_id = ?`, chatJID, userID,
		)
	}
}

// ── AFK ───────────────────────────────────────────────────────────────────────

// AFKStatus holds an active AFK entry.
type AFKStatus struct {
	Message string
	SetAt   time.Time
}

func getAFK(userID string) *AFKStatus {
	var msg string
	var setAt int64
	err := settingsDB.QueryRow(
		`SELECT message, set_at FROM afk_status WHERE user_id = ?`, userID,
	).Scan(&msg, &setAt)
	if err != nil {
		return nil
	}
	return &AFKStatus{Message: msg, SetAt: time.Unix(setAt, 0)}
}

func setAFK(userID, message string) {
	settingsDB.Exec(
		`INSERT INTO afk_status (user_id, message, set_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET message = excluded.message, set_at = excluded.set_at`,
		userID, message, time.Now().Unix(),
	)
}

func clearAFK(userID string) {
	settingsDB.Exec(`DELETE FROM afk_status WHERE user_id = ?`, userID)
}

// ── filters ───────────────────────────────────────────────────────────────────

func getFilters(scope, chatJID string) map[string]string {
	rows, err := settingsDB.Query(
		`SELECT keyword, response FROM filters WHERE scope = ? AND chat_jid = ?`, scope, chatJID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			m[k] = v
		}
	}
	return m
}

func setFilter(scope, chatJID, keyword, response string) {
	settingsDB.Exec(
		`INSERT INTO filters (scope, chat_jid, keyword, response) VALUES (?, ?, ?, ?)
		 ON CONFLICT(scope, chat_jid, keyword) DO UPDATE SET response = excluded.response`,
		scope, chatJID, keyword, response,
	)
}

func delFilter(scope, chatJID, keyword string) bool {
	res, err := settingsDB.Exec(
		`DELETE FROM filters WHERE scope = ? AND chat_jid = ? AND keyword = ?`, scope, chatJID, keyword,
	)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

func matchFilter(scope, chatJID, text string) (response string, found bool) {
	rows, err := settingsDB.Query(
		`SELECT keyword, response FROM filters WHERE scope = ? AND chat_jid = ?`, scope, chatJID,
	)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	lower := strings.ToLower(text)
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			if strings.Contains(lower, strings.ToLower(k)) {
				return v, true
			}
		}
	}
	return "", false
}

// ── antistatus ────────────────────────────────────────────────────────────────

func getAntistatusEnabled(chatJID string) bool {
	var enabled int
	settingsDB.QueryRow(`SELECT enabled FROM antistatus_settings WHERE chat_jid = ?`, chatJID).Scan(&enabled)
	return enabled == 1
}

func setAntistatusEnabled(chatJID string, on bool) {
	v := 0
	if on {
		v = 1
	}
	settingsDB.Exec(
		`INSERT INTO antistatus_settings (chat_jid, enabled) VALUES (?, ?)
		 ON CONFLICT(chat_jid) DO UPDATE SET enabled = excluded.enabled`,
		chatJID, v,
	)
}
