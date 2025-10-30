package dto

type CoverageCreateDTO struct {
	CodeArea    string  `json:"code_area" validate:"required"`
	Name        string  `json:"name" validate:"required"`
	Address     string  `json:"address"`
	Description string  `json:"description"`
	RangeArea   int     `json:"range_area"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type CoverageUpdateDTO struct {
	CodeArea    *string  `json:"code_area"`
	Name        *string  `json:"name"`
	Address     *string  `json:"address"`
	Description *string  `json:"description"`
	RangeArea   *int     `json:"range_area"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}
