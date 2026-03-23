package alertdialog

import (
	"crypto/rand"
	"fmt"
)

func randomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("adlg-%x", b)
}
