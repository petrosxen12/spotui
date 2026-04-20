package ui

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/petrosxen/spotui/internal/app"
)

type localPlayerStatus struct {
	supported       bool
	binaryAvailable bool
	process         string
	device          string
	message         string
}

func (s localPlayerStatus) visible() bool {
	return s.supported && (s.process != "" || s.device != "" || s.message != "" || !s.binaryAvailable)
}

func (s localPlayerStatus) statusLine() string {
	if !s.supported {
		return ""
	}

	state := s.process
	if state == "" {
		if s.binaryAvailable {
			state = "ready"
		} else {
			state = "unavailable"
		}
	}

	line := "Local player: " + state
	if s.device != "" {
		line += "  ·  " + s.device
	}
	if s.message != "" {
		line += "  ·  " + s.message
	}
	return line
}

func (s localPlayerStatus) statusTone() string {
	switch {
	case !s.supported:
		return ""
	case !s.binaryAvailable:
		return "error"
	case s.process == "running":
		return "success"
	case s.process == "starting":
		return "info"
	case s.process == "unhealthy":
		return "error"
	case s.message != "":
		return "info"
	default:
		return "subtle"
	}
}

func fetchLocalPlayerStatusCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		status, err := getLocalPlayerStatus(service)
		return localPlayerStatusMsg{status: status, err: err}
	}
}

func localPlayerStatusActionCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		status, err := getLocalPlayerStatus(service)
		if err != nil {
			return localPlayerActionMsg{text: "Failed to fetch local player status", err: err}
		}
		return localPlayerActionMsg{
			text:   formatLocalPlayerAction("Fetched local player status", status),
			status: status,
		}
	}
}

func startLocalPlayerCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		err := callServiceContextMethod(service, "StartLocalPlayer")
		if err != nil {
			return localPlayerActionMsg{text: "Failed to start local player", err: err}
		}
		status, statusErr := getLocalPlayerStatus(service)
		if statusErr != nil {
			return localPlayerActionMsg{text: "Started local player", err: nil}
		}
		return localPlayerActionMsg{text: formatLocalPlayerAction("Started local player", status), status: status, err: nil}
	}
}

func stopLocalPlayerCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		err := callServiceContextMethod(service, "StopLocalPlayer")
		if err != nil {
			return localPlayerActionMsg{text: "Failed to stop local player", err: err}
		}
		status, statusErr := getLocalPlayerStatus(service)
		if statusErr != nil {
			return localPlayerActionMsg{text: "Stopped local player", err: nil}
		}
		return localPlayerActionMsg{text: formatLocalPlayerAction("Stopped local player", status), status: status, err: nil}
	}
}

func useLocalPlayerCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		err := callServiceContextMethod(service, "UseLocalPlayer")
		if err != nil {
			return localPlayerActionMsg{text: "Failed to select local player", err: err}
		}
		status, statusErr := getLocalPlayerStatus(service)
		if statusErr != nil {
			return localPlayerActionMsg{text: "Selected local player", err: nil}
		}
		return localPlayerActionMsg{text: formatLocalPlayerAction("Selected local player", status), status: status, err: nil}
	}
}

func resetLocalPlayerCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		err := callServiceContextMethod(service, "ResetLocalPlayer")
		if err != nil {
			return localPlayerActionMsg{text: "Failed to reset local player", err: err}
		}
		status, statusErr := getLocalPlayerStatus(service)
		if statusErr != nil {
			return localPlayerActionMsg{text: "Reset local player", err: nil}
		}
		return localPlayerActionMsg{text: formatLocalPlayerAction("Reset local player", status), status: status, err: nil}
	}
}

func getLocalPlayerStatus(service app.PlayerService) (localPlayerStatus, error) {
	method, ok := serviceContextMethod(service, "LocalPlayerStatus")
	if !ok {
		return localPlayerStatus{}, nil
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(context.Background())})
	if len(results) != 2 {
		return localPlayerStatus{}, fmt.Errorf("LocalPlayerStatus returned %d values, want 2", len(results))
	}

	status := localPlayerStatus{supported: true}
	if err := errorFromResult(results[1]); err != nil {
		return status, err
	}

	if field := exportedField(results[0], "Binary"); field.IsValid() {
		status.binaryAvailable = extractBinaryAvailable(field)
	}
	if field := exportedField(results[0], "Process"); field.IsValid() {
		status.process = extractStateString(field, "State", "Status")
	}
	if field := exportedField(results[0], "Device"); field.IsValid() {
		status.device = extractStateString(field, "Name", "ID")
	}
	if field := exportedField(results[0], "Message"); field.IsValid() {
		status.message = extractStateString(field, "Text", "Message")
	}

	return status, nil
}

func formatLocalPlayerAction(prefix string, status localPlayerStatus) string {
	suffix := localPlayerSummary(status)
	if suffix == "" {
		return prefix
	}
	return prefix + ". " + suffix
}

func localPlayerSummary(status localPlayerStatus) string {
	if !status.supported {
		return ""
	}

	parts := make([]string, 0, 3)
	state := status.process
	if state == "" {
		if status.binaryAvailable {
			state = "ready"
		} else {
			state = "unavailable"
		}
	}
	parts = append(parts, "Current state: "+state)
	if status.device != "" {
		parts = append(parts, "device "+status.device)
	}
	if status.message != "" {
		parts = append(parts, status.message)
	}
	return joinLocalPlayerSummary(parts)
}

func joinLocalPlayerSummary(parts []string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " · ")
}

func callServiceContextMethod(service app.PlayerService, name string) error {
	method, ok := serviceContextMethod(service, name)
	if !ok {
		return fmt.Errorf("%s is not available in this build", name)
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(context.Background())})
	if len(results) != 1 {
		return fmt.Errorf("%s returned %d values, want 1", name, len(results))
	}
	return errorFromResult(results[0])
}

func serviceContextMethod(service app.PlayerService, name string) (reflect.Value, bool) {
	if service == nil {
		return reflect.Value{}, false
	}

	method := reflect.ValueOf(service).MethodByName(name)
	if !method.IsValid() {
		return reflect.Value{}, false
	}
	methodType := method.Type()
	if methodType.NumIn() != 1 || !methodType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return reflect.Value{}, false
	}
	return method, true
}

func errorFromResult(value reflect.Value) error {
	if !value.IsValid() || value.IsNil() {
		return nil
	}
	err, _ := value.Interface().(error)
	return err
}

func exportedField(value reflect.Value, name string) reflect.Value {
	value = indirectValue(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return value.FieldByName(name)
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() {
		switch value.Kind() {
		case reflect.Interface, reflect.Pointer:
			if value.IsNil() {
				return reflect.Value{}
			}
			value = value.Elem()
		default:
			return value
		}
	}
	return value
}

func extractBinaryAvailable(value reflect.Value) bool {
	value = indirectValue(value)
	if !value.IsValid() {
		return false
	}
	if value.Kind() == reflect.Bool {
		return value.Bool()
	}
	if value.Kind() == reflect.Struct {
		for _, name := range []string{"Available", "Present", "Found"} {
			field := value.FieldByName(name)
			if field.IsValid() && field.Kind() == reflect.Bool {
				return field.Bool()
			}
		}
	}
	return false
}

func extractStateString(value reflect.Value, nestedNames ...string) string {
	value = indirectValue(value)
	if !value.IsValid() {
		return ""
	}

	if value.Kind() == reflect.String {
		return value.String()
	}

	if value.Kind() == reflect.Bool {
		if value.Bool() {
			return "running"
		}
		return "stopped"
	}

	if value.Kind() == reflect.Struct {
		for _, name := range nestedNames {
			field := value.FieldByName(name)
			if str := extractStateString(field); str != "" {
				return str
			}
		}
		if field := value.FieldByName("Running"); field.IsValid() && field.Kind() == reflect.Bool {
			return extractStateString(field)
		}
	}

	return ""
}
