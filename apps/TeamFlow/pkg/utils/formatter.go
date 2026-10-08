package utils

import "strconv"

func FormatUintToID(userID uint) string {
	return strconv.FormatUint(uint64(userID), 10)
}

func StringToUint(userID string) (uint, error) {
	id, err := strconv.ParseUint(userID, 10, 10)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func FormatIDToUint(userID string) (uint, error) {
	return StringToUint(userID)
}

func FormatUintToString(userID uint) string {
	return strconv.FormatUint(uint64(userID), 10)
}
