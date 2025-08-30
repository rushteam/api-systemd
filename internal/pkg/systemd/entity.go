package systemd

type Unit struct {
	Service       string `json:"service"`
	Description   string `json:"description"`
	LoadState     string `json:"load_state"`
	ActiveState   string `json:"active_state"`
	UnitFileState string `json:"unit_file_state"`
	PID           int    `json:"pid"`
}
