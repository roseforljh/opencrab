package controller

import (
	"net/http"

	"github.com/QuantumNous/opencrab/model"
	"github.com/QuantumNous/opencrab/service"
	"github.com/QuantumNous/opencrab/setting"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	groupsMap := make(map[string]struct{})
	for _, modelName := range model.GetEnabledModels() {
		for _, groupName := range model.GetModelEnableGroups(modelName) {
			groupsMap[groupName] = struct{}{}
		}
	}
	groupNames := make([]string, 0, len(groupsMap))
	for groupName := range groupsMap {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	groupSet := make(map[string]struct{})
	for _, modelName := range model.GetEnabledModels() {
		for _, groupName := range model.GetModelEnableGroups(modelName) {
			groupSet[groupName] = struct{}{}
		}
	}
	for groupName := range groupSet {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": 1,
				"desc":  desc,
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
