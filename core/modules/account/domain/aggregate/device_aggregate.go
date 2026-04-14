package aggregate

import (
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"
)

var ErrDeviceAggregateNotInitialized = errors.New("device aggregate is not initialized")

type DeviceAggregate struct {
	device *entity.Device
}

func NewDeviceAggregate(deviceID string) (*DeviceAggregate, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, stackErr.Error(ErrDeviceAggregateNotInitialized)
	}

	return &DeviceAggregate{
		device: &entity.Device{ID: deviceID},
	}, nil
}

func (a *DeviceAggregate) Restore(snapshot *entity.Device) error {
	if snapshot == nil {
		return stackErr.Error(ErrDeviceAggregateNotInitialized)
	}

	cloned := *snapshot
	cloned.DeviceName = utils.ClonePtr(snapshot.DeviceName)
	cloned.OSName = utils.ClonePtr(snapshot.OSName)
	cloned.OSVersion = utils.ClonePtr(snapshot.OSVersion)
	cloned.AppVersion = utils.ClonePtr(snapshot.AppVersion)
	cloned.UserAgent = utils.ClonePtr(snapshot.UserAgent)
	cloned.LastIPAddress = utils.ClonePtr(snapshot.LastIPAddress)
	cloned.LastSeenAt = utils.ClonePtr(snapshot.LastSeenAt)
	a.device = &cloned
	return nil
}

func (a *DeviceAggregate) Register(
	accountID string,
	registration entity.DeviceRegistration,
	now time.Time,
) error {
	if a == nil || a.device == nil || strings.TrimSpace(a.device.ID) == "" {
		return stackErr.Error(ErrDeviceAggregateNotInitialized)
	}

	device, err := entity.NewDevice(a.device.ID, accountID, registration, now)
	if err != nil {
		return stackErr.Error(err)
	}

	a.device = device
	return nil
}

func (a *DeviceAggregate) RefreshRegistration(registration entity.DeviceRegistration, now time.Time) error {
	if a == nil || a.device == nil {
		return stackErr.Error(ErrDeviceAggregateNotInitialized)
	}

	return stackErr.Error(a.device.RefreshRegistration(registration, now))
}

func (a *DeviceAggregate) Touch(userAgent, ipAddress string, now time.Time) error {
	if a == nil || a.device == nil {
		return stackErr.Error(ErrDeviceAggregateNotInitialized)
	}

	a.device.Touch(userAgent, ipAddress, now)
	return nil
}

func (a *DeviceAggregate) Snapshot() (*entity.Device, error) {
	if a == nil || a.device == nil {
		return nil, stackErr.Error(ErrDeviceAggregateNotInitialized)
	}

	cloned := *a.device
	cloned.DeviceName = utils.ClonePtr(a.device.DeviceName)
	cloned.OSName = utils.ClonePtr(a.device.OSName)
	cloned.OSVersion = utils.ClonePtr(a.device.OSVersion)
	cloned.AppVersion = utils.ClonePtr(a.device.AppVersion)
	cloned.UserAgent = utils.ClonePtr(a.device.UserAgent)
	cloned.LastIPAddress = utils.ClonePtr(a.device.LastIPAddress)
	cloned.LastSeenAt = utils.ClonePtr(a.device.LastSeenAt)
	return &cloned, nil
}

func (a *DeviceAggregate) DeviceID() string {
	if a == nil || a.device == nil {
		return ""
	}

	return a.device.ID
}
