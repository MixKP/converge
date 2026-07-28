package branch

type Branch struct {
	ID       int    `json:"id"       example:"1"`
	Name     string `json:"name"     example:"Siam"`
	Capacity int    `json:"capacity" example:"30"`
}

type UpdateCapacityRequest struct {
	Capacity int `json:"capacity" example:"30"`
}
