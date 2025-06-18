package internal

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
	Reason  string `json:"reason"`
}
