// This is not actually well-formed Go (there's a syntax error near the end).  Do not change the
// next comment line.

//builtin-etags

package Pack //D |package Pack|

const  C1, C2 = 10, 20 //D |const  C1|
 const C3 = 10 // Not tagged, not at start of line
const (
	C4 = 10 // Not tagged, inside list
)

var V1, V2 int //D |var V1|
var (
	V3 int // Not tagged, inside list
)

type T1[T any] struct {} //D |type T1|
type (
	T2 = int // Not tagged, inside list
)

func F1(x int) { } //D |func F1|
func (self *t3) M1(y int) { } //D |func (self *t3) M1|

func F2[T any](x int) { //D |func F2|
	var lv1 int
	const lc1 = 10
	type lt1 = int
}

func bad() { ++x } //D |func bad|
