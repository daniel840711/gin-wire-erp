package model

type RequestLog struct {
	RequestID   string `bson:"request_id" json:"request_id"`
	Path        string `bson:"path" json:"path"`
	Method      string `bson:"method" json:"method"`
	ProjectName string `bson:"project_name,omitempty" json:"project_name,omitempty"`
	Body        string `bson:"body,omitempty" json:"body,omitempty"`
	IPHash      string `bson:"ip_hash,omitempty" json:"ip_hash,omitempty"`
	UserAgent   string `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	Version     string `bson:"version,omitempty" json:"version,omitempty"`
	RequestTS   string `bson:"request_ts" json:"request_ts"`
	LoggedAt    string `bson:"logged_at" json:"logged_at"`
}
