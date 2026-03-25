package model

import (
	"errors"
	"sort"

	"github.com/roseforljh/opencrab/common"
)

func GetGroupModels(group string) []string {
	if group == "" {
		return []string{}
	}
	if common.MemoryCacheEnabled {
		channelSyncLock.RLock()
		defer channelSyncLock.RUnlock()
		if group2model2channels == nil {
			return []string{}
		}
		models := make([]string, 0, len(group2model2channels[group]))
		for modelName := range group2model2channels[group] {
			models = append(models, modelName)
		}
		sort.Strings(models)
		return models
	}
	var rows []struct{ Model string }
	_ = DB.Table("abilities").Select("distinct model").Where(commonGroupCol+" = ? and enabled = ?", group, true).Scan(&rows).Error
	models := make([]string, 0, len(rows))
	for _, row := range rows {
		models = append(models, row.Model)
	}
	sort.Strings(models)
	return models
}

func IsModelInGroup(modelName string, group string) bool {
	if modelName == "" || group == "" {
		return false
	}
	for _, item := range GetGroupModels(group) {
		if item == modelName {
			return true
		}
	}
	return false
}

func GetRandomEnabledChannel(modelName string, channelType int) (*Channel, error) {
	group := "default"
	channel, err := GetRandomSatisfiedChannel(group, modelName, 0)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, errors.New("no enabled channel found")
	}
	if channelType != 0 && channel.Type != channelType {
		return nil, errors.New("no enabled channel found for requested type")
	}
	return channel, nil
}

func GetEnabledChannelsByModel(modelName string) ([]*Channel, error) {
	if modelName == "" {
		return []*Channel{}, nil
	}
	var channels []*Channel
	err := DB.Table("channels").
		Select("channels.*").
		Joins("join abilities on abilities.channel_id = channels.id").
		Where("abilities.model = ? and abilities.enabled = ? and channels.status = ?", modelName, true, common.ChannelStatusEnabled).
		Group("channels.id").
		Find(&channels).Error
	return channels, err
}
