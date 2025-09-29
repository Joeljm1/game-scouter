package application

import (
	"testing"
)

func TestSetAndGetUser(t *testing.T) {
	// user1 := &data.User{}
	// users := []*data.User{user1, data.AnonymousUser()}
	// app := Application{}
	// for i, user := range users {
	//
	// 	r, err := http.NewRequest(http.MethodGet, "/", nil)
	// 	if err != nil {
	// 		t.Fatalf("%v) Err making req. Err: %v", i, err.Error())
	// 	}
	//
	// 	newReq := app.SetUser(r, user, "tokentest")
	// 	userFromCtx := app.GetUser(newReq)
	// 	if userFromCtx != user {
	// 		t.Errorf("%v) Err getting same user.", i)
	// 	}
	// 	tokenFromtx := app.GetTok(newReq)
	// 	if tokenFromtx != "tokentest" {
	// 		t.Errorf("%v) Err getting same token.", i)
	// 	}
	// }
}
