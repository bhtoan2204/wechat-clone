package entity

import (
	"errors"
	"strings"
	"time"

	"go-socket/core/shared/pkg/stackErr"
)

type DeviceType string

const (
	DeviceTypeWeb     DeviceType = "web"
	DeviceTypeIOS     DeviceType = "ios"
	DeviceTypeAndroid DeviceType = "android"
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeOther   DeviceType = "other"
)

var ErrInvalidDevice = errors.New("invalid device")

type DeviceRegistration struct {
	DeviceUID  string
	DeviceName string
	DeviceType string
	OSName     string
	OSVersion  string
	AppVersion string
	UserAgent  string
	IPAddress  string
}

type Device struct {
	ID            string
	AccountID     string
	DeviceUID     string
	DeviceName    *string
	DeviceType    DeviceType
	OSName        *string
	OSVersion     *string
	AppVersion    *string
	UserAgent     *string
	LastIPAddress *string
	LastSeenAt    *time.Time
	IsTrusted     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewDevice(id, accountID string, registration DeviceRegistration, now time.Time) (*Device, error) {
	id = strings.TrimSpace(id)
	accountID = strings.TrimSpace(accountID)
	if id == "" || accountID == "" {
		return nil, stackErr.Error(ErrInvalidDevice)
	}

	deviceType, err := normalizeDeviceType(registration.DeviceType)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	deviceUID := strings.TrimSpace(registration.DeviceUID)
	if deviceUID == "" {
		return nil, stackErr.Error(ErrInvalidDevice)
	}

	normalizedNow := now.UTC()
	device := &Device{
		ID:         id,
		AccountID:  accountID,
		DeviceUID:  deviceUID,
		DeviceType: deviceType,
		CreatedAt:  normalizedNow,
		UpdatedAt:  normalizedNow,
	}

	if err := device.RefreshRegistration(registration, normalizedNow); err != nil {
		return nil, stackErr.Error(err)
	}

	return device, nil
}

func (d *Device) RefreshRegistration(registration DeviceRegistration, now time.Time) error {
	if d == nil {
		return stackErr.Error(ErrInvalidDevice)
	}

	deviceUID := strings.TrimSpace(registration.DeviceUID)
	if deviceUID == "" {
		return stackErr.Error(ErrInvalidDevice)
	}
	if d.DeviceUID != "" && d.DeviceUID != deviceUID {
		return stackErr.Error(ErrInvalidDevice)
	}

	deviceType, err := currentOrIncomingDeviceType(d.DeviceType, registration.DeviceType)
	if err != nil {
		return stackErr.Error(err)
	}

	d.DeviceUID = deviceUID
	d.DeviceType = deviceType
	d.DeviceName = mergeOptionalString(d.DeviceName, registration.DeviceName)
	d.OSName = mergeOptionalString(d.OSName, registration.OSName)
	d.OSVersion = mergeOptionalString(d.OSVersion, registration.OSVersion)
	d.AppVersion = mergeOptionalString(d.AppVersion, registration.AppVersion)

	d.Touch(registration.UserAgent, registration.IPAddress, now)
	return nil
}

func (d *Device) Touch(userAgent, ipAddress string, now time.Time) {
	if d == nil {
		return
	}

	if next := normalizeOptionalString(userAgent); next != nil {
		d.UserAgent = next
	}
	if next := normalizeOptionalString(ipAddress); next != nil {
		d.LastIPAddress = next
	}

	normalizedNow := now.UTC()
	d.LastSeenAt = &normalizedNow
	d.UpdatedAt = normalizedNow
	if d.CreatedAt.IsZero() {
		d.CreatedAt = normalizedNow
	}
}

func normalizeDeviceType(value string) (DeviceType, error) {
	switch DeviceType(strings.ToLower(strings.TrimSpace(value))) {
	case "", DeviceTypeWeb:
		return DeviceTypeWeb, nil
	case DeviceTypeIOS:
		return DeviceTypeIOS, nil
	case DeviceTypeAndroid:
		return DeviceTypeAndroid, nil
	case DeviceTypeDesktop:
		return DeviceTypeDesktop, nil
	case DeviceTypeOther:
		return DeviceTypeOther, nil
	default:
		return "", stackErr.Error(ErrInvalidDevice)
	}
}

func currentOrIncomingDeviceType(current DeviceType, incoming string) (DeviceType, error) {
	if strings.TrimSpace(incoming) == "" && current != "" {
		return current, nil
	}
	return normalizeDeviceType(incoming)
}

func mergeOptionalString(current *string, incoming string) *string {
	if next := normalizeOptionalString(incoming); next != nil {
		return next
	}
	return current
}

func normalizeOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	cloned := value
	return &cloned
}
