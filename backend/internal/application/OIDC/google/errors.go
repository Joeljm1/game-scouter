package google

import "fmt"

type AuthError struct {
	Err error
	Msg string
}

func (err AuthError) Error() string {
	return fmt.Sprintf("error:%v , Msg:%v", err.Err.Error(), err.Msg)
}

type ReqError struct {
	Err error
	Msg string
}

func (err ReqError) Error() string {
	return fmt.Sprintf("error:%v , Msg:%v", err.Err.Error(), err.Msg)
}
