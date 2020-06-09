package core

// FilterOkOrInfoHistory ...
func FilterOkOrInfoHistory(history History) History {
	var h History
	for _, op := range history {
		if op.Type == OpTypeOk || op.Type == OpTypeInfo {
			h = append(h, op)
		}
	}
	return h
}

// FilterOutNemesisHistory ...
func FilterOutNemesisHistory(history History) History {
	var h History
	for _, op := range history {
		if op.Process.Present() && op.Process.MustGet() == NemesisProcessMagicNumber {
			continue
		}
		h = append(h, op)
	}
	return h
}

// FilterOkHistory ...
func FilterOkHistory(history History) History {
	var h History
	for _, v := range history {
		if v.Type == OpTypeOk {
			h = append(h, v)
		}
	}
	return h
}

// FilterFailedHistory ...
func FilterFailedHistory(history History) History {
	var h History
	for _, v := range history {
		if v.Type == OpTypeFail {
			h = append(h, v)
		}
	}
	return h
}
