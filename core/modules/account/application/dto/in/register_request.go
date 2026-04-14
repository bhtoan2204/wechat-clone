// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type RegisterRequest struct {
	DisplayName string `json:"display_name" form:"display_name" binding:"required"`
	Email       string `json:"email" form:"email" binding:"required,email"`
	Password    string `json:"password" form:"password" binding:"required"`
	DeviceUid   string `json:"device_uid" form:"device_uid" binding:"required"`
	DeviceName  string `json:"device_name" form:"device_name"`
	DeviceType  string `json:"device_type" form:"device_type"`
	OsName      string `json:"os_name" form:"os_name"`
	OsVersion   string `json:"os_version" form:"os_version"`
	AppVersion  string `json:"app_version" form:"app_version"`
	UserAgent   string `json:"user_agent" form:"user_agent"`
	IpAddress   string `json:"ip_address" form:"ip_address"`
}

func (r *RegisterRequest) Normalize() {
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)
	r.DeviceUid = strings.TrimSpace(r.DeviceUid)
	r.DeviceName = strings.TrimSpace(r.DeviceName)
	r.DeviceType = strings.TrimSpace(r.DeviceType)
	r.OsName = strings.TrimSpace(r.OsName)
	r.OsVersion = strings.TrimSpace(r.OsVersion)
	r.AppVersion = strings.TrimSpace(r.AppVersion)
	r.UserAgent = strings.TrimSpace(r.UserAgent)
	r.IpAddress = strings.TrimSpace(r.IpAddress)
}

func (r *RegisterRequest) Validate() error {
	r.Normalize()
	if r.DisplayName == "" {
		return stackErr.Error(errors.New("display_name is required"))
	}
	if r.Email == "" {
		return stackErr.Error(errors.New("email is required"))
	}
	if r.Password == "" {
		return stackErr.Error(errors.New("password is required"))
	}
	if r.DeviceUid == "" {
		return stackErr.Error(errors.New("device_uid is required"))
	}
	return nil
}
