package common

import "strconv"

func ChannelTypeId(raw string) int {
	id, _ := strconv.Atoi(raw)
	return id
}
