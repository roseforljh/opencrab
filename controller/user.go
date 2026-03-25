package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/roseforljh/opencrab/common"
	"github.com/roseforljh/opencrab/i18n"
	"github.com/roseforljh/opencrab/model"
	"github.com/roseforljh/opencrab/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type PinLoginRequest struct {
	Pin string `json:"pin"`
}

type UpdatePinRequest struct {
	CurrentPin string `json:"current_pin"`
	Pin        string `json:"pin"`
	ConfirmPin string `json:"confirm_pin"`
}

type UpdateSelfRequest struct {
	Username         string `json:"username"`
	DisplayName      string `json:"display_name"`
	Password         string `json:"password"`
	OriginalPassword string `json:"original_password"`
}

func PinLogin(c *gin.Context) {
	var req PinLoginRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if req.Pin == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	user := model.GetRootUser()
	if user == nil {
		common.ApiErrorI18n(c, i18n.MsgUserNotExists)
		return
	}
	if user.Status != common.UserStatusEnabled || !user.ValidatePin(req.Pin) {
		c.JSON(http.StatusOK, gin.H{
			"message": "PIN 错误或用户已被封禁",
			"success": false,
		})
		return
	}

	setupLogin(user, c)
}

func UpdateSelfPin(c *gin.Context) {
	var req UpdatePinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if req.Pin == "" || req.ConfirmPin == "" || req.Pin != req.ConfirmPin {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if len(req.Pin) < 4 || len(req.Pin) > 12 {
		common.ApiError(c, errors.New("PIN 长度需在4到12位之间"))
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.PinHash != "" && !user.ValidatePin(req.CurrentPin) {
		common.ApiError(c, errors.New("当前 PIN 错误"))
		return
	}
	if err := user.SetPin(req.Pin); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := user.Update(false); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUpdateFailed)
		return
	}

	common.ApiSuccessI18n(c, i18n.MsgUpdateSuccess, nil)
}

func setupLogin(user *model.User, c *gin.Context) {
	session := sessions.Default(c)
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	session.Set("group", user.Group)
	err := session.Save()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserSessionSaveFailed)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
		"data": map[string]any{
			"id":           user.Id,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"role":         user.Role,
			"status":       user.Status,
			"group":        user.Group,
		},
	})
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
	})
}

func GenerateAccessToken(c *gin.Context) {
	id := c.GetInt("id")
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	randI := common.GetRandomInt(4)
	key, err := common.GenerateRandomKey(29 + randI)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgGenerateFailed)
		common.SysLog("failed to generate key: " + err.Error())
		return
	}
	user.SetAccessToken(key)

	if model.DB.Where("access_token = ?", user.AccessToken).First(user).RowsAffected != 0 {
		common.ApiErrorI18n(c, i18n.MsgUuidDuplicate)
		return
	}

	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user.AccessToken,
	})
}

func GetSelf(c *gin.Context) {
	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user.Remark = ""

	responseData := map[string]interface{}{
		"id":            user.Id,
		"username":      user.Username,
		"display_name":  user.DisplayName,
		"role":          user.Role,
		"status":        user.Status,
		"email":         user.Email,
		"group":         user.Group,
		"request_count": user.RequestCount,
		"setting":       user.Setting,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    responseData,
	})
}

func GetUserModels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		id = c.GetInt("id")
	}
	user, err := model.GetUserCache(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groups := service.GetUserUsableGroups(user.Group)
	var models []string
	for group := range groups {
		for _, g := range model.GetGroupEnabledModels(group) {
			if !common.StringsContains(models, g) {
				models = append(models, g)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
}

func UpdateSelf(c *gin.Context) {
	var updateReq UpdateSelfRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&updateReq); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	validateUser := model.User{
		Username:    updateReq.Username,
		DisplayName: updateReq.DisplayName,
		Password:    updateReq.Password,
	}
	if validateUser.Password == "" {
		validateUser.Password = "$I_LOVE_U"
	}
	if err := common.Validate.Struct(&validateUser); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidInput)
		return
	}

	cleanUser := model.User{
		Id:          c.GetInt("id"),
		Username:    updateReq.Username,
		Password:    updateReq.Password,
		DisplayName: updateReq.DisplayName,
	}
	if cleanUser.Password == "$I_LOVE_U" {
		cleanUser.Password = ""
	}
	updatePassword, err := checkUpdatePassword(updateReq.OriginalPassword, cleanUser.Password, cleanUser.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := cleanUser.Update(updatePassword); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func checkUpdatePassword(originalPassword string, newPassword string, userId int) (updatePassword bool, err error) {
	currentUser, err := model.GetUserById(userId, true)
	if err != nil {
		return
	}

	if !common.ValidatePasswordAndHash(originalPassword, currentUser.Password) && currentUser.Password != "" {
		err = fmt.Errorf("原密码错误")
		return
	}
	if newPassword == "" {
		return
	}
	updatePassword = true
	return
}

func DeleteSelf(c *gin.Context) {
	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)

	if user.Role == common.RoleRootUser {
		common.ApiErrorI18n(c, i18n.MsgUserCannotDeleteRootUser)
		return
	}

	err := model.DeleteUserById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
