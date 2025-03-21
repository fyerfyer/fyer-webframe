package orm

// Op 定义操作符的类型和行为
type Op struct {
	Type    OpType
	Keyword string
}

type OpType uint8

const (
	OpBinary  OpType = iota // 二元运算符 e.g., =, >, <
	OpUnary                 // 一元运算符 e.g., NOT, EXISTS
	OpTernary               // 三元运算符 e.g., BETWEEN
)

// 预定义操作符
var (
	opEQ         = Op{Type: OpBinary, Keyword: "="}
	opGT         = Op{Type: OpBinary, Keyword: ">"}
	opNOT        = Op{Type: OpUnary, Keyword: "NOT"}
	opISNULL     = Op{Type: OpUnary, Keyword: "IS NULL"}
	opNOTNULL    = Op{Type: OpUnary, Keyword: "IS NOT NULL"}
	opLTE        = Op{Type: OpBinary, Keyword: "<="}
	opLT         = Op{Type: OpBinary, Keyword: "<"}
	opGTE        = Op{Type: OpBinary, Keyword: ">="}
	opLIKE       = Op{Type: OpBinary, Keyword: "LIKE"}
	opNOTLIKE    = Op{Type: OpBinary, Keyword: "NOT LIKE"}
	opIN         = Op{Type: OpBinary, Keyword: "IN"}
	opNOTIN      = Op{Type: OpBinary, Keyword: "NOT IN"}
	opBETWEEN    = Op{Type: OpTernary, Keyword: "BETWEEN"}
	opNOTBETWEEN = Op{Type: OpTernary, Keyword: "NOT BETWEEN"}
)
