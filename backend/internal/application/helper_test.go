package application

import (
	"game-scouter-api/internal/data"
	"net/http"
	"testing"
)

func TestSetAndGetUser(t *testing.T) {
	user1 := &data.User{}
	users := []*data.User{user1, data.AnonymousUser()}
	app := Application{}
	for i, user := range users {

		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("%v) Err making req. Err: %v", i, err.Error())
		}

		newReq := app.SetUser(r, user)
		userFromCtx := app.GetUser(newReq)
		if userFromCtx != user {
			t.Errorf("%v) Err getting same user.", i)
		}
	}
}
