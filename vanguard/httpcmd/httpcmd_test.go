package httpcmd

import (
	"fmt"
	"testing"
	"time"

	ut "cement/unittest"
)

type AddCmd struct {
	Num int
}

func (a *AddCmd) String() string {
	return fmt.Sprintf("addcmd with num %v\n", a.Num)
}

type DecCmd struct {
}

func (a *DecCmd) String() string {
	return fmt.Sprintf("deccmd\n")
}

type unknownCmd struct {
}

func (a *unknownCmd) String() string {
	return fmt.Sprintf("unknown cmd \n")
}

type NumberAddService struct {
	Num int
}

func (s *NumberAddService) HandleTask(t *Task) *TaskResult {
	switch t.Cmds[0].(type) {
	case *AddCmd:
		c, _ := t.Cmds[0].(*AddCmd)
		s.Num += c.Num
		return t.SucceedWithResult(s.Num)
	case *DecCmd:
		s.Num -= 1
		return t.SucceedWithResult(s.Num)
	default:
		panic("shouldn't be here")
	}
}

func (s *NumberAddService) SupportedCmds() []Command {
	return []Command{&AddCmd{}, &DecCmd{}}
}

func TestAddNumberService(t *testing.T) {
	e := &EndPoint{
		Name: "addnum",
		IP:   "127.0.0.1",
		Port: 5555,
	}

	s := &NumberAddService{
		Num: 0,
	}

	server_is_ok := make(chan int)
	go func() {
		server_is_ok <- 1
		if err := Run(s, e); err != nil {
			panic("fun service failed:" + err.Error())
		}
	}()

	<-server_is_ok
	<-time.After(time.Second)

	task := NewTask()
	task.AddCmd(&AddCmd{Num: 1})
	proxy, err := GetProxy(e, s.SupportedCmds())
	ut.Assert(t, err == nil, "create proxy should succeed")

	var num int
	err = proxy.HandleTask(task, &num)
	ut.Equal(t, err, (*Error)(nil))
	ut.Equal(t, num, 1)

	err = proxy.HandleTask(task, &num)
	ut.Equal(t, err, (*Error)(nil))
	ut.Equal(t, num, 2)

	task.ClearCmd()

	task.AddCmd(&unknownCmd{})
	err = proxy.HandleTask(task, nil)
	ut.Assert(t, err != nil, "handle add task should succeed but get %v", err)
	ut.Equal(t, err.(*Error).Info, ErrUnknownCmd.Info)

	task.ClearCmd()
	task.AddCmd(&DecCmd{})
	proxy.HandleTask(task, &num)
	ut.Equal(t, num, 1)
}
