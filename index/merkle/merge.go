package merkle

import (
	"encoding/json"
	"github.com/treeverse/lakefs/logging"
	"strings"

	"github.com/treeverse/lakefs/index/model"
)

func CompareEntries(a, b *model.Entry) (eqs int) {
	// names first
	eqs = strings.Compare(a.GetName(), b.GetName())
	// directories second
	if eqs == 0 && a.EntryType != b.EntryType {
		if a.EntryType < b.EntryType {
			eqs = -1
		} else if a.EntryType > b.EntryType {
			eqs = 1
		} else {
			eqs = 0
		}
	}
	return
}

func prettyEntryNames(entries []*model.Entry) string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.GetName()
	}
	data, err := json.Marshal(names)
	if err != nil {
		return "unknown"
	}
	return string(data)
}

func prettyWorkspaceNames(changes []*model.WorkspaceEntry) string {
	names := make([]string, len(changes))
	for i, change := range changes {
		names[i] = change.Entry().GetName()
	}
	data, err := json.Marshal(names)
	if err != nil {
		return "unknown"
	}
	return string(data)
}

func mergeChanges(current []*model.Entry, changes []*model.WorkspaceEntry, logger logging.Logger) ([]*model.Entry, error) {
	merged := make([]*model.Entry, 0)
	nextCurrent := 0
	nextChange := 0
	for {
		// if both lists still have values, compare
		if nextChange < len(changes) && nextCurrent < len(current) {
			currEntry := current[nextCurrent]
			currChange := changes[nextChange]
			comparison := CompareEntries(currEntry, currChange.Entry())
			if comparison == 0 {
				// this is an override or deletion - do nothing

				// overwrite
				if !currChange.Tombstone {
					merged = append(merged, currChange.Entry())
				}
				// otherwise, skip both
				nextCurrent++
				nextChange++
			} else if comparison == -1 {
				nextCurrent++
				// current entry comes first
				merged = append(merged, currEntry)
			} else {
				nextChange++
				// changed entry comes first
				if currChange.Tombstone {
					logger.
						WithField("current_change_name", currChange.GetName()).
						WithField("current_entry_name", currEntry.GetName()).
						WithField("changes", prettyWorkspaceNames(changes)).
						WithField("entries", prettyEntryNames(current)).
						Error("trying to remove an entry that does not exist")
				} else {
					merged = append(merged, currChange.Entry())
				}
			}
		} else if nextChange < len(changes) {
			// only changes left
			currChange := changes[nextChange]
			if currChange.Tombstone {
				logger.
					WithField("current_change_name", currChange.GetName()).
					WithField("changes", prettyWorkspaceNames(changes)).
					WithField("entries", prettyEntryNames(current)).
					Error("trying to remove an entry that does not exist, no entries are left")
			} else {
				merged = append(merged, currChange.Entry())
			}
			nextChange++
		} else if nextCurrent < len(current) {
			// only current entries left
			currEntry := current[nextCurrent]
			merged = append(merged, currEntry)
			nextCurrent++
		} else {
			// done with both
			break
		}
	}
	return merged, nil
}