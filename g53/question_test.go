package g53

import (
	"g53/util"
	"testing"
)

func matchQuestionRaw(t *testing.T, rawData string, q *Question) {
	wire, _ := util.HexStrToBytes(rawData)
	buffer := util.NewInputBuffer(wire)
	nq, err := QuestionFromWire(buffer)
	Assert(t, err == nil, "wire data is valid")
	matchQuestion(t, nq, q)
	render := NewMsgRender()
	nq.Rend(render)
	WireMatch(t, render.Data(), wire)
}

func matchQuestion(t *testing.T, nq *Question, q *Question) {
	Assert(t, nq.Name.Equals(q.Name), "name should equal")
	Equal(t, nq.Type, q.Type)
	Equal(t, nq.Class, q.Class)
}

func TestQuestionFromToWire(t *testing.T) {
	n, _ := NameFromString("foo.example.com.")
	matchQuestionRaw(t, "03666f6f076578616d706c6503636f6d0000020001", &Question{
		Name:  n,
		Type:  RR_NS,
		Class: CLASS_IN,
	})

	n, _ = NameFromString("bar.example.com.")
	matchQuestionRaw(t, "03626172076578616d706c6503636f6d0000010003", &Question{
		Name:  n,
		Type:  RR_A,
		Class: CLASS_CH,
	})

	n, _ = NameFromString("test.example.com.")
	matchQuestionRaw(t, "0474657374076578616d706c6503636f6d0000010001", &Question{
		Name:  n,
		Type:  RR_A,
		Class: CLASS_IN,
	})
}
