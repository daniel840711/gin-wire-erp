package model

type ResponseLog struct {
	// 對應鍵
	RequestID   string `bson:"request_id" json:"request_id"`
	ProjectName string `bson:"project_name,omitempty" json:"project_name,omitempty"`
	Code        int    `bson:"code" json:"code"`
	StatusCode  int    `bson:"status_code" json:"status_code"`
	Body        string `bson:"body,omitempty" json:"body,omitempty"`
	Error       string `bson:"error,omitempty" json:"error,omitempty"`
	Version     string `bson:"version,omitempty" json:"version,omitempty"`
	ResponseTS  string `bson:"response_ts" json:"response_ts"`
	LoggedAt    string `bson:"logged_at" json:"logged_at"`
}
