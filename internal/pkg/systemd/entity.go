package systemd

type Unit struct {
	Name          string `json:"name"`
	Service       string `json:"service"`
	Description   string `json:"description"`
	LoadState     string `json:"load_state"`
	ActiveState   string `json:"active_state"`
	SubState      string `json:"sub_state"`
	UnitFileState string `json:"unit_file_state"`
	PID           int    `json:"pid"`
}
