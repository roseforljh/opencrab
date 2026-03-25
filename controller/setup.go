package controller

import (
	"time"

	"github.com/roseforljh/opencrab/common"
	"github.com/roseforljh/opencrab/constant"
	"github.com/roseforljh/opencrab/model"
	"github.com/gin-gonic/gin"
)

type Setup struct {
	Status   bool `json:"status"`
	RootInit bool `json:"root_init"`
}

type SetupRequest struct {
	Pin        string `json:"pin"`
	ConfirmPin string `json:"confirmPin"`
}

func GetSetup(c *gin.Context) {
	setup := Setup{
		Status: constant.Setup,
	}
	if constant.Setup {
		c.JSON(200, gin.H{
			"success": true,
			"data":    setup,
		})
		return
	}
	setup.RootInit = model.RootUserExists()
	c.JSON(200, gin.H{
		"success": true,
		"data":    setup,
	})
}

func PostSetup(c *gin.Context) {
	// Check if setup is already completed
	if constant.Setup {
		c.JSON(200, gin.H{
			"success": false,
			"message": "系统已经初始化完成",
		})
		return
	}

	// Check if root user already exists
	rootExists := model.RootUserExists()

	var req SetupRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": "请求参数有误",
		})
		return
	}

	// If root doesn't exist, validate and create admin account
	if !rootExists {
		if req.Pin == "" {
			c.JSON(200, gin.H{
				"success": false,
				"message": "PIN 不能为空",
			})
			return
		}
		if req.Pin != req.ConfirmPin {
			c.JSON(200, gin.H{
				"success": false,
				"message": "两次输入的 PIN 不一致",
			})
			return
		}
		if len(req.Pin) < 4 || len(req.Pin) > 12 {
			c.JSON(200, gin.H{
				"success": false,
				"message": "PIN 长度需在4到12位之间",
			})
			return
		}

		hashedPassword, err := common.Password2Hash(req.Pin)
		if err != nil {
			c.JSON(200, gin.H{
				"success": false,
				"message": "系统错误: " + err.Error(),
			})
			return
		}
		rootUser := model.User{
			Username:    "root",
			Password:    hashedPassword,
			Role:        common.RoleRootUser,
			Status:      common.UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: nil,
			Quota:       100000000,
		}
		if err := rootUser.SetPin(req.Pin); err != nil {
			c.JSON(200, gin.H{
				"success": false,
				"message": "设置 PIN 失败: " + err.Error(),
			})
			return
		}
		err = model.DB.Create(&rootUser).Error
		if err != nil {
			c.JSON(200, gin.H{
				"success": false,
				"message": "创建管理员账号失败: " + err.Error(),
			})
			return
		}
	}

	// Update setup status
	constant.Setup = true

	setup := model.Setup{
		Version:       common.Version,
		InitializedAt: time.Now().Unix(),
	}
	err = model.DB.Create(&setup).Error
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": "系统初始化失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "系统初始化成功",
	})
}
