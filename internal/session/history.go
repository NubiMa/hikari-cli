package session

import (
	"sort"
	"time"
)

// HistoryGroup groups sessions by a date label (Today, Yesterday, date string).
type HistoryGroup struct {
	Label    string
	Sessions []ListEntry
}

// GroupByDate organises a list of sessions into date-labelled groups.
// The list is assumed to already be sorted by UpdatedAt descending.
func GroupByDate(entries []ListEntry) []HistoryGroup {
	now := time.Now()
	today := truncateToDay(now)
	yesterday := today.AddDate(0, 0, -1)

	groupMap := make(map[string][]ListEntry)
	var groupOrder []string
	seen := make(map[string]bool)

	for _, e := range entries {
		day := truncateToDay(e.UpdatedAt)
		var label string
		switch {
		case day.Equal(today):
			label = "Today"
		case day.Equal(yesterday):
			label = "Yesterday"
		default:
			label = day.Format("2 January 2006")
		}

		if !seen[label] {
			seen[label] = true
			groupOrder = append(groupOrder, label)
		}
		groupMap[label] = append(groupMap[label], e)
	}

	groups := make([]HistoryGroup, 0, len(groupOrder))
	for _, label := range groupOrder {
		sessions := groupMap[label]
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
		})
		groups = append(groups, HistoryGroup{Label: label, Sessions: sessions})
	}
	return groups
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
