/* Do not reformat this one, see gotags_test.go for instructions.  There are literal tabs in the comments. */
package  main  //D |package  main|

 type t1 = int //D | type t1|
type (
	t2 = bool //D |	t2|
	t3 struct { } //D |	t3|
)
type t4[T any] struct { } //D |type t4|

const c1 = 5 //D |const c1|
const (
	c2, c3 = c1, 8 //D |	c2|	c2, c3|
)
const c4, c5 = 10, 20 //D |const c4|const c4, c5|

var v1 int //D |var v1|
var (
	v2 bool //D |	v2|
	v4, v5 int //D |	v4|	v4, v5|
)
var v6, v7 int //D |var v6|var v6, v7|

var v8 int; var v9 int //D |var v8|var v8 int; var v9|

func f1(x int) { } //D |func f1|
func (self *t3) m1(y int) { } //D |func (self *t3) m1|

func f2[T any](x int) { //D |func f2|
	var lv1 int
	const lc1 = 10
	type lt1 = int
}

type i1 interface {				//D |type i1|
	if1(x int) int 				//D |	if1|
	if2(y int) int				//D |	if2|
}
