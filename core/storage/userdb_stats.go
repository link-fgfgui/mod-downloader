package storage

import "time"

func (s *userStore) incrementUsageStat(key UsageStatKey, delta int64) error {
	_, err := s.db.Exec(
		`INSERT INTO usage_stats(key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
			value = usage_stats.value + excluded.value,
			updated_at = excluded.updated_at`,
		string(key), delta, time.Now().Unix(),
	)
	return err
}

func (s *userStore) usageStatValues() (map[UsageStatKey]int64, error) {
	rows, err := s.db.Query(`SELECT key, value FROM usage_stats`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make(map[UsageStatKey]int64)
	for rows.Next() {
		var key UsageStatKey
		var value int64
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
