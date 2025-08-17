package auth

import (
	"game-scouter-api/internal/application"
	"testing"
)

func TestGetDataMap(t *testing.T) {
	app := application.Application{}
	authApp := AuthApplication{&app}
	codeOgSLice := []string{"123456789", "987654321", "abcd123"}
	map1 := map[string]any{
		"code": codeOgSLice,
	}
	data, err := authApp.SerializeGoB(map1)
	if err != nil {
		t.Fatal(err.Error())
	}
	retMap, err := app.GetSessDataMap(data)
	if err != nil {
		t.Fatal(err.Error())
	}
	codeAny, ok := retMap["code"]
	if !ok {
		t.Fatal("code is not present after deserialising")
	}
	codeSlice, ok := codeAny.([]string)
	if !ok {
		t.Fatal("val of key cod is not str slice")
	}
	for i := range codeOgSLice {
		if codeOgSLice[i] != codeSlice[i] {
			t.Errorf("%v) Expected:%v, Got:%v", i+1, codeOgSLice[i], codeSlice[i])
		}
	}
}
