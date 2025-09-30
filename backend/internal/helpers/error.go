package helpers

import "fmt"

type Err struct {
	Msg string
	Err error
}

func (err Err) Error() string {
	return fmt.Sprintf("Msg:%v,Err:%v", err.Msg, err.Err)
}
