package model

type Document struct {
	ID          string                 `json:"id"`
	Category    string                 `json:"category"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Keywords    []string               `json:"keywords"`
	Deeplink    string                 `json:"deeplink"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
