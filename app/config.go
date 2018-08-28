package app

// AppConfig is a dandelion app config structure.
type AppConfig struct {
	ID          int64  `db:"id" json:"id"`
	AppID       string `db:"app_id" json:"app_id"`
	Status      int    `db:"status" json:"status"`
	Version     string `db:"version" json:"version"`
	Host        string `db:"host" json:"host"`
	InstanceID  string `db:"instance_id" json:"instance_id"`
	CommitID    string `db:"commit_id" json:"commit_id"`
	MD5Sum      string `db:"md5sum" json:"md5sum"`
	Author      string `db:"author" json:"author"`
	CreatedTime int64  `db:"created_time" json:"created_time"`
	UpdatedTime int64  `db:"updated_time" json:"updated_time"`
}

// Status is a dandelion app instance status structure
type Status struct {
	ID          int64  `json:"-" db:"id"`
	AppID       string `json:"app_id" db:"app_id"`
	Host        string `json:"host" db:"host"`
	InstanceID  string `json:"instance_id" db:"instance_id"`
	ConfigID    int64  `json:"config_id,omitempty" db:"config_id"`
	CommitID    string `json:"commit_id,omitempty" db:"commit_id"`
	Status      int    `json:"status" db:"status"`
	CreatedTime int64  `json:"created_time,omitempty" db:"created_time"`
	UpdatedTime int64  `json:"updated_time,omitempty" db:"updated_time"`
}

// ClientConfig is client app config
type ClientConfig struct {
	ID         int
	AppID      string
	Host       string
	InstanceID string
	Version    string
}
