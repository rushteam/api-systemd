package systemd

import (
	"fmt"

	"github.com/godbus/dbus"
)

const destBus = "org.freedesktop.systemd1"
const objectPath = "/org/freedesktop/systemd1"
const getMethod = "org.freedesktop.DBus.Properties.Get"
const mngerMethod = "org.freedesktop.systemd1.Manager"
const destUnit = "org.freedesktop.systemd1.Unit"
const destService = "org.freedesktop.systemd1.Service"

// Load unit from systemd
func Load(serviceName string) (*Unit, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	var path dbus.ObjectPath
	err = conn.Object(destBus, objectPath).Call(mngerMethod+".GetUnit", 0, serviceName).Store(&path)
	if err != nil {
		return nil, fmt.Errorf("failed to get object path: %w", err)
	}

	var desc string
	err = conn.Object(destBus, path).Call(getMethod, 0, destUnit, "Description").Store(&desc)
	if err != nil {
		return nil, fmt.Errorf("failed to get description: %w", err)
	}

	var loadStatus string
	err = conn.Object(destBus, path).Call(getMethod, 0, destUnit, "LoadState").Store(&loadStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get load state: %w", err)
	}

	var activeStatus string
	err = conn.Object(destBus, path).Call(getMethod, 0, destUnit, "ActiveState").Store(&activeStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get active state: %w", err)
	}

	var unitFileState string
	err = conn.Object(destBus, path).Call(getMethod, 0, destUnit, "UnitFileState").Store(&unitFileState)
	if err != nil {
		return nil, fmt.Errorf("failed to get unit file status: %w", err)
	}

	var pid int
	err = conn.Object(destBus, path).Call(getMethod, 0, destService, "MainPID").Store(&pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get main pid: %w", err)
	}

	u := &Unit{
		Service:       serviceName,
		Description:   desc,
		LoadState:     loadStatus,
		ActiveState:   activeStatus,
		UnitFileState: unitFileState,
		PID:           pid,
	}

	return u, nil
}

// Send an action to systemd
func Send(serviceName string, action string, mode string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	var path dbus.ObjectPath
	obj := conn.Object(destBus, objectPath)

	switch action {
	case "start":
		err = obj.Call(mngerMethod+".StartUnit", 0, serviceName, mode).Store(&path)
	case "restart":
		err = obj.Call(mngerMethod+".RestartUnit", 0, serviceName, mode).Store(&path)
	case "stop":
		err = obj.Call(mngerMethod+".StopUnit", 0, serviceName, mode).Store(&path)
	case "reload":
		err = obj.Call(mngerMethod+".ReloadUnit", 0, serviceName, mode).Store(&path)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	if err != nil {
		return fmt.Errorf("failed to execute action %s on service %s: %w", action, serviceName, err)
	}
	return nil
}

// EnableUnit 启用服务单元
func EnableUnit(serviceName string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object(destBus, objectPath)

	// EnableUnitFiles 方法的参数: files, runtime, force
	files := []string{serviceName}
	runtime := false
	force := false

	var carries bool
	var changes []interface{}
	err = obj.Call(mngerMethod+".EnableUnitFiles", 0, files, runtime, force).Store(&carries, &changes)
	if err != nil {
		return fmt.Errorf("failed to enable unit %s: %w", serviceName, err)
	}

	return nil
}

// DisableUnit 禁用服务单元
func DisableUnit(serviceName string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object(destBus, objectPath)

	// DisableUnitFiles 方法的参数: files, runtime
	files := []string{serviceName}
	runtime := false

	var changes []interface{}
	err = obj.Call(mngerMethod+".DisableUnitFiles", 0, files, runtime).Store(&changes)
	if err != nil {
		return fmt.Errorf("failed to disable unit %s: %w", serviceName, err)
	}

	return nil
}

// ReloadDaemon 重新加载systemd守护进程
func ReloadDaemon() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object(destBus, objectPath)
	err = obj.Call(mngerMethod+".Reload", 0).Err
	if err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

// GetServiceStatusText 获取服务状态文本（类似systemctl status输出）
func GetServiceStatusText(serviceName string) (string, error) {
	unit, err := Load(serviceName)
	if err != nil {
		return "", err
	}

	// 构建类似systemctl status的输出
	status := fmt.Sprintf("● %s - %s\n", serviceName, unit.Description)
	status += fmt.Sprintf("   Loaded: %s (%s)\n", unit.LoadState, unit.UnitFileState)
	status += fmt.Sprintf("   Active: %s", unit.ActiveState)

	if unit.PID > 0 {
		status += fmt.Sprintf(" (pid: %d)", unit.PID)
	}
	status += "\n"

	return status, nil
}

// CheckSystemdAvailable 检查systemd是否可用
func CheckSystemdAvailable() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object(destBus, objectPath)

	// 尝试获取systemd版本
	var version string
	err = obj.Call("org.freedesktop.DBus.Properties.Get", 0, mngerMethod, "Version").Store(&version)
	if err != nil {
		return fmt.Errorf("systemd not available: %w", err)
	}

	return nil
}
