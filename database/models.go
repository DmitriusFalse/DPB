package database

type Pack struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type File struct {
	ID              int    `json:"id"`
	PackID          int    `json:"pack_id"`
	FileName        string `json:"file_name"`
	CategoryID      int    `json:"category_id"`
	CategoryName    string `json:"category_name"`
	SubcategoryName string `json:"subcategory_name"`
	FileHash        string `json:"file_hash"`
	LastSynced      string `json:"last_synced"`
}

type Tag struct {
	ID              int    `json:"id"`
	FileID          int    `json:"file_id"`
	PackID          int    `json:"pack_id"`
	TagName         string `json:"tag_name"`
	CategoryName    string `json:"category_name"`
	SubcategoryName string `json:"subcategory_name"`
	Aliases         string `json:"aliases"`
}

type SavedPrompt struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	PositiveText string `json:"positive_text"`
	NegativeText string `json:"negative_text"`
	IsFavorite   bool   `json:"is_favorite"`
	CreatedAt    string `json:"created_at"`
}

type TagPreset struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	PositiveTags string `json:"positive_tags"`
	NegativeTags string `json:"negative_tags"`
}

type SubcategoryInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
