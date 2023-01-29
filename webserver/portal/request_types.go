package portal

type ListUserRequest struct {
	SortType  int    `schema:"sortType"`
	Sort      int    `schema:"sort"`
	KeySearch string `schema:"keySearch"`
	Limit     int    `schema:"limit"`
	Offset    int    `schema:"offset"`
}
