package diagnostic

type Version struct {
	Major   int32 `json:"major"`
	Minor   int32 `json:"minor"`
	Release int32 `json:"release"`
	Build   int32 `json:"build"`
}

type Document struct {
	Name         string `json:"name"`
	Directory    string `json:"directory"`
	Path         string `json:"path"`
	TypeID       int32  `json:"type_id"`
	Changed      bool   `json:"changed"`
	ReadOnly     bool   `json:"read_only"`
	HasSavedPath bool   `json:"has_saved_path"`
}

type Report struct {
	Connected bool      `json:"connected"`
	ProgID    string    `json:"prog_id"`
	Version   *Version  `json:"version"`
	Document  *Document `json:"document"`
	State     string    `json:"state"`
}
