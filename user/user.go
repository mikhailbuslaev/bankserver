package user

type User struct {
	Id string `json:"id"`
	AuthKey string
	Balance float64 `json:"balance"`
}

func Create(id, authKey string,balance float64) *User{
	u := &User{Id:id, AuthKey: authKey, Balance:balance}
	return u
}