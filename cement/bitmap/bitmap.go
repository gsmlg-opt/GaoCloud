package bitmap

var (
	tA = [8]byte{1, 2, 4, 8, 16, 32, 64, 128}
	tB = [8]byte{254, 253, 251, 247, 239, 223, 191, 127}
)

func NewSlice(l int) []byte {
	remainder := l % 8
	if remainder != 0 {
		remainder = 1
	}
	return make([]byte, l/8+remainder)
}

func Get(m []byte, i int) bool {
	return m[i/8]&tA[i%8] != 0
}

func Set(m []byte, i int, v bool) {
	index := i / 8
	bit := i % 8
	if v {
		m[index] = m[index] | tA[bit]
	} else {
		m[index] = m[index] & tB[bit]
	}
}

func Len(m []byte) int {
	return len(m) * 8
}

type Bitmap []byte

func New(l int) Bitmap {
	return NewSlice(l)
}

func (b Bitmap) Len() int {
	return Len(b)
}

func (b Bitmap) Get(i int) bool {
	return Get(b, i)
}

func (b Bitmap) Set(i int, v bool) {
	Set(b, i, v)
}
