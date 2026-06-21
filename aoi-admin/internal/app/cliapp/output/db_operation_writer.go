package output

import (
	"fmt"
	"io"
)

// WriteDBOperationResult 将 db 子命令执行结果写入命令输出流。
func WriteDBOperationResult(w io.Writer, message, sql string, printSQL bool) {
	fmt.Fprintln(w, message)
	if printSQL {
		fmt.Fprintln(w, sql)
	}
}
